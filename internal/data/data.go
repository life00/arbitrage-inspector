package data

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func FetchData(exchanges models.Exchanges, currencies models.Currencies) error {
	if false {
		err := errors.New("something went wrong")
		slog.Error("something went wrong")
		return err
	}
	fmt.Println(exchanges, currencies)
	return nil
}
