package data

import (
	"log/slog"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateInput(exchanges []string, currencies []string) ([]ccxt.IExchange, error) {
	slog.Debug("validating exchanges...")
	err := validateExchanges(exchanges)
	if err != nil {
		return nil, err
	}
	slog.Debug("initializing ccxt...")
	clients, err := loadCcxt(exchanges)
	if err != nil {
		return nil, err
	}
	// slog.Debug("identifying common currencies...")
	// commonCurrencies := getCommonValidCurrencies(&clients)

	// slog.Debug("validating currencies...")
	// err = validateCurrencies(currencies, commonCurrencies)
	// if err != nil {
	// 	return nil, err
	// }

	return clients, nil
}

// func getMarkets(ccxtExchangesPtr *[]ccxt.IExchange, currencies models.Currencies) models.Markets {
// 	slog.Debug("identifying common markets...")
// 	commonMarkets := getCommonValidMarkets(ccxtExchangesPtr)
//
// 	// find all possible currency pairs (available in the found common markets) based on input currencies
// 	slog.Debug("deriving input markets...")
// 	markets := getMatchingMarkets(commonMarkets, currencies)
//
// 	return markets
// }

func InitializeData(exchanges []string, currencies []string) (models.Exchanges, []ccxt.IExchange, error) {
	slog.Info("validating inputs...")
	clients, err := validateInput(exchanges, currencies)
	if err != nil {
		return nil, nil, err
	}

	// slog.Info("identifying markets...")
	// markets := getMarkets(&ccxtExchanges, currencies)

	return models.Exchanges{}, clients, nil
}
