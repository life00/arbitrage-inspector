package data

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
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

	slog.Debug(fmt.Sprintf("updating exchange data for %s...", exchangeId))

	var wg sync.WaitGroup
	var exchangeMu sync.Mutex

	errChan := make(chan error, 3)

	runTask := func(task func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := task(); err != nil {
				errChan <- err
			}
		}()
	}

	runTask(func() error {
		return updatePrices(clientPtr, &exchange, &exchangeMu)
	})

	if updateCurrencyFees {
		runTask(func() error {
			return updateCurrencies(clientPtr, &exchange, &exchangeMu)
		})
	}

	if updateMarketFees {
		runTask(func() error {
			return updateMarkets(clientPtr, &exchange, &exchangeMu)
		})
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	mu.Lock()
	(*exchanges)[exchangeId] = exchange
	mu.Unlock()

	return nil
}

// updateCurrencies fetches price data to update conversion prices
func updatePrices(clientPtr *ccxt.IExchange, exchange *models.Exchange, exchangeMu *sync.Mutex) error {
	client := *clientPtr

	exchangeMu.Lock()
	markets := make(map[string]models.Market, len(exchange.Markets))
	maps.Copy(markets, exchange.Markets)
	exchangeMu.Unlock()

	tickers, err := client.FetchTickers()
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	if len(tickers.Tickers) == 0 {
		return nil
	}

	for id, market := range markets {
		if ticker, ok := tickers.Tickers[id]; ok {
			if ticker.Bid != nil {
				if market.Bid, err = decimal.NewFromFloat64(*ticker.Bid); err != nil {
					return fmt.Errorf("invalid bid value for %s: %w", id, err)
				}
			}
			if ticker.Ask != nil {
				if market.Ask, err = decimal.NewFromFloat64(*ticker.Ask); err != nil {
					return fmt.Errorf("invalid ask value for %s: %w", id, err)
				}
			}
			if ticker.Timestamp != nil {
				market.Timestamp = time.UnixMilli(*ticker.Timestamp)
			}
			markets[id] = market
		}
	}

	exchangeMu.Lock()
	exchange.Markets = markets
	exchangeMu.Unlock()

	return nil
}

// updateCurrencies fetches currency data to update withdrawal fees and network details
func updateCurrencies(clientPtr *ccxt.IExchange, exchange *models.Exchange, exchangeMu *sync.Mutex) error {
	client := *clientPtr

	exchangeMu.Lock()
	currencies := make(map[string]models.Currency, len(exchange.Currencies))
	maps.Copy(currencies, exchange.Currencies)
	exchangeMu.Unlock()

	apiCurrencies, err := client.FetchCurrencies()
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	if len(apiCurrencies.Currencies) == 0 {
		return nil
	}

	for id, currency := range currencies {
		if apiCurrency, ok := apiCurrencies.Currencies[id]; ok {

			currency.Networks = make(map[string]models.CurrencyNetwork)

			for name, network := range apiCurrency.Networks {
				if network.Active != nil && *network.Active && network.Withdraw != nil && *network.Withdraw &&
					network.Deposit != nil && *network.Deposit && network.Fee != nil {

					fee, err := decimal.NewFromFloat64(*network.Fee)
					if err != nil {
						slog.Warn(fmt.Sprintf("invalid fee for currency %s on network %s: %v", id, name, err))
						continue // Skip this network if fee is invalid
					}
					currency.Networks[name] = models.CurrencyNetwork{
						Id:            name,
						WithdrawalFee: fee,
					}
				}
			}

			currency.Id = id
			currencies[id] = currency
		}
	}

	exchangeMu.Lock()
	exchange.Currencies = currencies
	exchangeMu.Unlock()

	return nil
}

// updateMarkets fetches market data to update taker and maker fees
func updateMarkets(clientPtr *ccxt.IExchange, exchange *models.Exchange, exchangeMu *sync.Mutex) error {
	client := *clientPtr

	exchangeMu.Lock()
	markets := make(map[string]models.Market, len(exchange.Markets))
	maps.Copy(markets, exchange.Markets)
	exchangeMu.Unlock()

	apiMarkets, err := client.FetchMarkets()
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}

	apiMarketsMap := make(map[string]ccxt.MarketInterface)
	for _, apiMarket := range apiMarkets {
		if apiMarket.Symbol != nil {
			apiMarketsMap[*apiMarket.Symbol] = apiMarket
		}
	}

	for id, market := range markets {
		if apiMarket, ok := apiMarketsMap[id]; ok {
			if apiMarket.Taker != nil {
				var err error
				if market.TakerFee, err = decimal.NewFromFloat64(*apiMarket.Taker); err != nil {
					return fmt.Errorf("invalid taker fee for %s: %w", id, err)
				}
			}

			if apiMarket.Maker != nil {
				var err error
				if market.MakerFee, err = decimal.NewFromFloat64(*apiMarket.Maker); err != nil {
					return fmt.Errorf("invalid maker fee for %s: %w", id, err)
				}
			}
			markets[id] = market
		}
	}

	exchangeMu.Lock()
	exchange.Markets = markets
	exchangeMu.Unlock()

	return nil
}
