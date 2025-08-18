package data

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func validateExchanges(exchanges models.Exchanges) error {
	if false {
		err := errors.New("something went wrong")

		slog.Error("something went wrong")
		return err
	}
	fmt.Println(ccxt.Exchanges)
	return nil
}
