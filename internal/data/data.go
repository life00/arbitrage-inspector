package data

import (
	"fmt"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateInput(exchanges models.Exchanges, currencies models.Currencies) ([]ccxt.IExchange, error) {
	err := validateExchanges(exchanges)
	if err != nil {
		return nil, err
	}
	ccxtExchanges, err := loadCcxt(exchanges)
	if err != nil {
		return nil, err
	}
	// fetchCommonCurrencies()
	// validateCurrencies()
	fmt.Println(ccxtExchanges)
	return nil, nil
}

func FetchData(exchanges models.Exchanges, currencies models.Currencies) error {
	ccxtExchanges, err := validateInput(exchanges, currencies)
	if err != nil {
		return err
	}
	fmt.Println(ccxtExchanges)
	return nil
}
