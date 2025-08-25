package data

import (
	"log/slog"
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
