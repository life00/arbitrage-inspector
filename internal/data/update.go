package data

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
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
		return updatePrices(clientPtr, &exchange)
	})

	if updateCurrencyFees {
		runTask(func() error {
			return updateCurrencies(clientPtr, &exchange)
		})
	}

	if updateMarketFees {
		runTask(func() error {
			return updateMarkets(clientPtr, &exchange)
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
func updatePrices(clientPtr *ccxt.IExchange, exchange *models.Exchange) error {
	client := *clientPtr
	tickers, err := client.FetchTickers()
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	if len(tickers.Tickers) == 0 {
		return nil
	}

	for symbol, ticker := range tickers.Tickers {
		if market, ok := exchange.Markets[symbol]; ok {
			if ticker.Bid != nil {
				if market.Bid, err = decimal.NewFromFloat64(*ticker.Bid); err != nil {
					return fmt.Errorf("invalid bid value for %s: %w", symbol, err)
				}
			}
			if ticker.Ask != nil {
				if market.Ask, err = decimal.NewFromFloat64(*ticker.Ask); err != nil {
					return fmt.Errorf("invalid ask value for %s: %w", symbol, err)
				}
			}
			if ticker.Timestamp != nil {
				market.Timestamp = time.UnixMilli(*ticker.Timestamp)
			}
			exchange.Markets[symbol] = market
		}
	}
	return nil
}

// updateCurrencies fetches currency data to update withdrawal fees
func updateCurrencies(clientPtr *ccxt.IExchange, exchange *models.Exchange) error {
	client := *clientPtr
	apiCurrencies, err := client.FetchCurrencies()
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	if len(apiCurrencies.Currencies) == 0 {
		return nil
	}

	for id, currency := range exchange.Currencies {
		if apiCurrency, ok := apiCurrencies.Currencies[id]; ok {
			minFee := math.MaxFloat64
			var bestFee *float64
			var bestNetwork string

			if len(apiCurrency.Networks) > 0 {
				for name, network := range apiCurrency.Networks {
					if network.Active != nil && *network.Active && network.Withdraw != nil && *network.Withdraw &&
						network.Deposit != nil && *network.Deposit && network.Fee != nil && *network.Fee < minFee {
						minFee = *network.Fee
						bestFee = network.Fee
						bestNetwork = name
					}
				}
			}

			if bestFee != nil {
				var err error
				if currency.WithdrawalFee, err = decimal.NewFromFloat64(*bestFee); err != nil {
					return fmt.Errorf("invalid fee for currency %s on network %s: %w", id, bestNetwork, err)
				}
				currency.Network = bestNetwork
				exchange.Currencies[id] = currency
			}
		}
	}
	return nil
}

// updateMarkets fetches market data to update taker and maker fees
func updateMarkets(clientPtr *ccxt.IExchange, exchange *models.Exchange) error {
	client := *clientPtr
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

	for symbol, market := range exchange.Markets {
		if apiMarket, ok := apiMarketsMap[symbol]; ok {
			if apiMarket.Taker != nil {
				var err error
				if market.TakerFee, err = decimal.NewFromFloat64(*apiMarket.Taker); err != nil {
					return fmt.Errorf("invalid taker fee for %s: %w", symbol, err)
				}
			}

			if apiMarket.Maker != nil {
				var err error
				if market.MakerFee, err = decimal.NewFromFloat64(*apiMarket.Maker); err != nil {
					return fmt.Errorf("invalid maker fee for %s: %w", symbol, err)
				}
			}
			exchange.Markets[symbol] = market
		}
	}
	return nil
}
