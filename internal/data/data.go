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
	slog.Debug("initializing ccxt...")
	ccxtExchanges, err := loadCcxt(exchanges)
	if err != nil {
		return nil, err
	}
	slog.Debug("identifying common currencies...")
	commonCurrencies := getCommonValidCurrencies(&ccxtExchanges)

	slog.Debug("validating currencies...")
	err = validateCurrencies(currencies, commonCurrencies)
	if err != nil {
		return nil, err
	}

	return ccxtExchanges, nil
}

func getMarkets(ccxtExchangesPtr *[]ccxt.IExchange, currencies models.Currencies) models.Markets {
	if ccxtExchangesPtr == nil {
		return models.Markets{}
	}
	// ccxtExchanges := *ccxtExchangesPtr

	commonMarkets := getCommonValidMarkets(ccxtExchangesPtr)

	fmt.Println(commonMarkets)

	// find all possible currency pairs (available in the found common markets) based on input currencies

	return models.Markets{}
}

func InitializeDataFetcher(exchanges models.Exchanges, currencies models.Currencies) ([]ccxt.IExchange, models.Markets, error) {
	slog.Info("validating inputs...")
	ccxtExchanges, err := validateInput(exchanges, currencies)
	if err != nil {
		return nil, models.Markets{}, err
	}

	slog.Info("identifying markets...")
	markets := getMarkets(&ccxtExchanges, currencies)

	return ccxtExchanges, markets, nil
}
