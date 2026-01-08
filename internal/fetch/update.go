package fetch

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func updateExchange(
	clientPtr *ccxt.IExchange,
	mu *sync.Mutex,
	exchanges *models.Exchanges,
	updateCurrencyFees bool,
	updateMarketFees bool,
) error {
	client := *clientPtr
	exchangeId := client.GetId()

	mu.Lock()
	exchange, ok := (*exchanges)[exchangeId]
	if !ok {
		mu.Unlock()
		return fmt.Errorf("exchange not found in data structure")
	}
	mu.Unlock()

	var wg sync.WaitGroup
	var errs sync.Map

	runTask := func(task func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := task(); err != nil {
				errs.Store(time.Now().UnixNano(), err)
			}
		}()
	}

	pricesChan := make(chan map[string]models.Market, 1)
	currenciesChan := make(chan map[string]models.Currency, 1)
	feesChan := make(chan map[string]models.Market, 1)

	runTask(func() error {
		prices, err := fetchPrices(clientPtr, &exchange)
		pricesChan <- prices
		return err
	})

	if updateCurrencyFees {
		runTask(func() error {
			currencies, err := fetchCurrencies(clientPtr, &exchange)
			currenciesChan <- currencies
			return err
		})
	}

	if updateMarketFees {
		runTask(func() error {
			fees, err := fetchFees(clientPtr, &exchange)
			feesChan <- fees
			return err
		})
	}

	wg.Wait()
	close(pricesChan)
	close(currenciesChan)
	close(feesChan)

	var allErrors []error
	errs.Range(func(key, value any) bool {
		allErrors = append(allErrors, value.(error))
		return true
	})
	if len(allErrors) > 0 {
		return errors.Join(allErrors...)
	}

	// merge the results

	updatedPrices := <-pricesChan
	updatedCurrencies := <-currenciesChan
	updatedMarketsFees := <-feesChan

	for id, priceUpdate := range updatedPrices {
		market := exchange.Markets[id]
		market.Bid = priceUpdate.Bid
		market.Ask = priceUpdate.Ask
		market.Timestamp = priceUpdate.Timestamp
		exchange.Markets[id] = market
	}

	for id, currencyUpdate := range updatedCurrencies {
		currency := exchange.Currencies[id]
		currency.Networks = currencyUpdate.Networks
		exchange.Currencies[id] = currency
	}

	for id, feeUpdate := range updatedMarketsFees {
		market := exchange.Markets[id]
		market.TakerFee = feeUpdate.TakerFee
		exchange.Markets[id] = market
	}

	mu.Lock()
	(*exchanges)[exchangeId] = exchange
	mu.Unlock()

	return nil
}

// fetchPrices fetches price data to update conversion prices
func fetchPrices(clientPtr *ccxt.IExchange, exchange *models.Exchange) (map[string]models.Market, error) {
	client := *clientPtr

	updatedPrices := make(map[string]models.Market)

	tickers, err := client.FetchTickers()
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	if len(tickers.Tickers) == 0 {
		return nil, nil
	}

	for id := range exchange.Markets {
		if ticker, ok := tickers.Tickers[id]; ok {
			var updatedPrice models.Market
			if ticker.Bid != nil {
				if updatedPrice.Bid, err = decimal.NewFromFloat64(*ticker.Bid); err != nil {
					return nil, fmt.Errorf("invalid bid value for %s: %w", id, err)
				}
			}
			if ticker.Ask != nil {
				if updatedPrice.Ask, err = decimal.NewFromFloat64(*ticker.Ask); err != nil {
					return nil, fmt.Errorf("invalid ask value for %s: %w", id, err)
				}
			}
			if ticker.Timestamp != nil {
				updatedPrice.Timestamp = time.UnixMilli(*ticker.Timestamp)
			}
			updatedPrices[id] = updatedPrice
		}
	}

	return updatedPrices, nil
}

// fetchCurrencies fetches currency data to update withdrawal fees and network details
func fetchCurrencies(clientPtr *ccxt.IExchange, exchange *models.Exchange) (map[string]models.Currency, error) {
	client := *clientPtr

	updatedCurrencies := make(map[string]models.Currency)

	apiCurrencies, err := client.FetchCurrencies()
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	if len(apiCurrencies.Currencies) == 0 {
		return nil, nil
	}

	for id := range exchange.Currencies {
		if apiCurrency, ok := apiCurrencies.Currencies[id]; ok {

			var updatedCurrency models.Currency

			updatedCurrency.Networks = make(map[string]models.CurrencyNetwork)

			for name, network := range apiCurrency.Networks {
				if network.Active != nil && *network.Active && network.Withdraw != nil && *network.Withdraw &&
					network.Deposit != nil && *network.Deposit && network.Fee != nil {

					fee, err := decimal.NewFromFloat64(*network.Fee)
					if err != nil {
						slog.Warn(fmt.Sprintf("invalid fee for currency %s on network %s: %v", id, name, err))
						continue // Skip this network if fee is invalid
					}
					uppercaseName := strings.ToUpper(name)
					updatedCurrency.Networks[uppercaseName] = models.CurrencyNetwork{
						Id:            uppercaseName,
						WithdrawalFee: fee,
					}
				}
			}

			updatedCurrency.Id = id
			updatedCurrencies[id] = updatedCurrency
		}
	}

	return updatedCurrencies, nil
}

// fetchFees fetches market data to update taker and maker fees
func fetchFees(clientPtr *ccxt.IExchange, exchange *models.Exchange) (map[string]models.Market, error) {
	client := *clientPtr

	updatedFees := make(map[string]models.Market)

	apiMarkets, err := client.FetchMarkets()
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	apiMarketsMap := make(map[string]ccxt.MarketInterface)
	for _, apiMarket := range apiMarkets {
		if apiMarket.Symbol != nil {
			apiMarketsMap[*apiMarket.Symbol] = apiMarket
		}
	}

	for id := range exchange.Markets {
		if apiMarket, ok := apiMarketsMap[id]; ok {
			var updatedFee models.Market
			if apiMarket.Taker != nil {
				var err error
				if updatedFee.TakerFee, err = decimal.NewFromFloat64(*apiMarket.Taker); err != nil {
					return nil, fmt.Errorf("invalid taker fee for %s: %w", id, err)
				}
			}

			updatedFees[id] = updatedFee
		}
	}

	return updatedFees, nil
}
