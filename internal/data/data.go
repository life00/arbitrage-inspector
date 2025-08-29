package data

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateInput(exchanges []string, currencies []string) (models.Clients, error) {
	slog.Debug("validating exchanges...")
	err := validateExchanges(exchanges)
	if err != nil {
		return nil, err
	}
	slog.Debug("initializing ccxt...")
	clients, err := loadClient(exchanges)
	if err != nil {
		return nil, err
	}

	slog.Debug("validating currencies...")
	err = validateCurrencies(currencies, &clients)
	if err != nil {
		return nil, err
	}

	return clients, nil
}

// createData orchestrates the concurrent processing of exchange clients
func createData(currencies []string, clientsPtr *models.Clients) models.Exchanges {
	if clientsPtr == nil || len(*clientsPtr) == 0 {
		return models.Exchanges{}
	}

	currencySet := make(map[string]struct{})
	for _, c := range currencies {
		currencySet[c] = struct{}{}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	exchanges := make(models.Exchanges)

	for _, client := range *clientsPtr {
		wg.Add(1)
		go createExchange(&client, currencySet, &wg, &mu, exchanges)
	}

	wg.Wait()
	return exchanges
}

func InitializeData(exchanges []string, currencies []string) (models.Exchanges, models.Clients, error) {
	slog.Info("validating inputs...")
	clients, err := validateInput(exchanges, currencies)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("creating data structure...")
	data := createData(currencies, &clients)

	return data, clients, nil
}

func UpdateData(
	exchangesPtr *models.Exchanges,
	clientsPtr *models.Clients,
	updateCurrencyFees bool,
	updateMarketFees bool,
) error {
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
		go func() {
			defer wg.Done()
			if err := updateExchange(&client, &mu, exchangesPtr, updateCurrencyFees, updateMarketFees); err != nil {
				errChan <- fmt.Errorf("[Exchange: %s] %w", client.GetId(), err)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("data update failed with %d error(s):\n- %s", len(errors), strings.Join(errors, "\n- "))
	}

	return nil
}
