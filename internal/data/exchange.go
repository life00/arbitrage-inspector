package data

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateExchanges(exchanges models.Exchanges) error {
	invalidExchanges := []string{}
	for _, exchange := range exchanges.Exchanges {
		found := false
		for _, ccxtExchange := range ccxt.Exchanges {
			if strings.EqualFold(exchange.Name, ccxtExchange) {
				found = true
				break
			}
		}
		if !found {
			invalidExchanges = append(invalidExchanges, exchange.Name)
		}
	}

	if len(invalidExchanges) > 0 {
		err := fmt.Errorf("invalid exchanges: %s", strings.Join(invalidExchanges, ", "))
		slog.Error(err.Error())
		return err
	}

	return nil
}

func loadCcxt(exchanges models.Exchanges) ([]ccxt.IExchange, error) {
	return []ccxt.IExchange{}, nil
}
