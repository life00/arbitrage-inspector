// Package fetch provides functions to fetch data using CCXT library.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ccxt/ccxt/go/v4/pro"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateInput(config models.Config) (models.Clients, error) {
	err := validateExchanges(config.Exchanges)
	if err != nil {
		return nil, err
	}
	clients, err := loadClient(config.Exchanges, config.Authenticate)
	if err != nil {
		return nil, err
	}

	if config.CurrencyInputMode == models.SpecifiedCurrencies {
		err = validateCurrencies(config.Currencies, &clients)
		if err != nil {
			return nil, err
		}
	}

	return clients, nil
}

// createExchanges orchestrates the concurrent processing of exchange clients
func createExchanges(config models.Config, clientsPtr *models.Clients) models.Exchanges {
	if clientsPtr == nil || len(*clientsPtr) == 0 {
		return models.Exchanges{}
	}

	// these common maps are created and passed to the function here to avoid redundant processing inside of createExchange
	currencySet := make(map[string]struct{}) // can be nil

	if config.CurrencyInputMode == models.SpecifiedCurrencies {
		for _, c := range config.Currencies {
			currencySet[c] = struct{}{}
		}
	}

	excludedCurrencySet := make(map[string]struct{})
	for _, c := range config.ExcludedCurrencies {
		excludedCurrencySet[c] = struct{}{}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	exchanges := make(models.Exchanges)

	for _, client := range *clientsPtr {
		wg.Add(1)
		go createExchange(&client, config.CurrencyInputMode, currencySet, excludedCurrencySet, &wg, &mu, exchanges)
	}

	wg.Wait()
	return exchanges
}

func InitializeExchanges(config models.Config) (models.Exchanges, models.Clients, error) {
	slog.Debug("initializing exchanges")
	clients, err := validateInput(config)
	if err != nil {
		return nil, nil, err
	}

	exchanges := createExchanges(config, &clients)

	return exchanges, clients, nil
}

func UpdateExchanges(
	exchangesPtr *models.Exchanges,
	clientsPtr *models.Clients,
	updateCurrencyFees bool,
	updateMarketFees bool,
	timeout time.Duration,
) error {
	slog.Debug("updating exchanges")
	if clientsPtr == nil || len(*clientsPtr) == 0 {
		return fmt.Errorf("list of clients is empty")
	}
	if exchangesPtr == nil || len(*exchangesPtr) == 0 {
		return fmt.Errorf("list of exchange data is empty")
	}
	if len(*clientsPtr) != len(*exchangesPtr) {
		return fmt.Errorf("length of clients and exchange data is not matching")
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(*clientsPtr))

	for _, client := range *clientsPtr {
		wg.Add(1)
		go func(c ccxtpro.IExchange) {
			defer wg.Done()

			// context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			if err := updateExchange(ctx, &c, &mu, exchangesPtr, updateCurrencyFees, updateMarketFees); err != nil {
				errChan <- fmt.Errorf("[Exchange: %s] %w", c.GetId(), err)
			}
		}(client)
	}

	wg.Wait()
	close(errChan)

	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("exchange data update failed with %d error(s):\n- %s", len(errors), strings.Join(errors, "\n- "))
	}

	return nil
}

// UpdateOrderBooks() updates orderbook data for specified markets in exchangesPtr
func UpdateOrderBooks(exchangesPtr *models.Exchanges, clientsPtr *models.Clients) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for exID, client := range *clientsPtr {
		ex, exists := (*exchangesPtr)[exID]
		if !exists {
			continue
		}

		for marketID := range ex.Markets {
			wg.Add(1)

			marketRef := ex.Markets[marketID]

			go func(c *ccxtpro.IExchange, eID, mID string, mkt models.Market) {
				defer wg.Done()

				ob, err := (*c).FetchOrderBook(mID, ccxtpro.WithFetchOrderBookLimit(100))

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					errs = append(errs, fmt.Errorf("[%s-%s]: %w", eID, mID, err))
					return
				}

				mkt.OrderBook = ob
				(*exchangesPtr)[eID].Markets[mID] = mkt
			}(&client, exID, marketID, marketRef)
		}
	}

	wg.Wait()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
