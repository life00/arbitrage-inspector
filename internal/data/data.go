package data

import (
	"fmt"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func FetchData(exchanges models.Exchanges, currencies models.Currencies) error {
	err := validateExchanges(exchanges)
	if err != nil {
		return err
	}

	fmt.Println(exchanges, currencies)
	return nil
}
