package data

import (
	"fmt"
	"log/slog"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateInput(exchanges models.Exchanges, currencies models.Currencies) ([]ccxt.IExchange, error) {
	slog.Debug("validating exchanges...")
	err := validateExchanges(exchanges)
	if err != nil {
		return nil, err
	}
	slog.Debug("initializing CCXT...")
	ccxtExchanges, err := loadCcxt(exchanges)
	if err != nil {
		return nil, err
	}
	// fetchCommonCurrencies()
	// validateCurrencies()
	fmt.Println(ccxtExchanges[0].FetchBalance())
	return nil, nil
}

func FetchData(exchanges models.Exchanges, currencies models.Currencies) error {
	slog.Info("validating inputs...")
	ccxtExchanges, err := validateInput(exchanges, currencies)
	if err != nil {
		return err
	}
	fmt.Println(ccxtExchanges)
	return nil
}
