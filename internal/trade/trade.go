package trade

import (
	"fmt"
	"log/slog"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func CalculateExpectedReturn(path models.TransactionPath, pairsPtr *models.Pairs) decimal.Decimal {
	pairs := *pairsPtr
	expectedReturn := decimal.One

	for _, pairKey := range path {
		pair, ok := pairs[pairKey]
		if !ok {
			slog.Error(fmt.Sprintf("pair not found for key: %+v\n", pairKey))
			return decimal.Decimal{}
		}

		expectedReturn, _ = expectedReturn.Mul(pair.Weight)
	}
	result, _ := expectedReturn.Sub(decimal.One)

	return result
}
