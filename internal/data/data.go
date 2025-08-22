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
	slog.Debug("identifying common currencies...")
	commonCurrencies := getCommonActiveCurrencies(&ccxtExchanges)

	slog.Debug("validating currencies...")
	err = validateCurrencies(currencies, commonCurrencies)
	if err != nil {
		return nil, err
	}

	return ccxtExchanges, nil
}

func getCurrencyPairs(ccxtExchangesPtr *[]ccxt.IExchange, currencies models.Currencies) models.CurrencyPairs {
	if ccxtExchangesPtr == nil {
		return models.CurrencyPairs{}
	}
	// ccxtExchanges := *ccxtExchangesPtr

	// extract a list of markets which are active, linear (I guess?), percentage fee, correct side of fee (?????)
	// find common markets across all exchanges (create a reusable function)
	// find all possible currency pairs (available in the found common markets) based on input currencies

	return models.CurrencyPairs{}
}

// TODO: it shouldn't be called fetching, because it is just initializing
// there should be a separate function which will take currency pairs and exchanges as input
func FetchData(exchanges models.Exchanges, currencies models.Currencies) error {
	slog.Info("validating inputs...")
	ccxtExchanges, err := validateInput(exchanges, currencies)
	if err != nil {
		return err
	}
	fmt.Println(ccxtExchanges)

	// currencyPairs := getCurrencyPairs(&ccxtExchanges, currencies)

	// fmt.Println(currencyPairs)
	return nil
}
