// Package trade provides functions to execute and manage trades using CCXT library.
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

// GetSimplePath() transforms a TransactionPath into a simpler string representation,
// where each string is formatted as "exchange:currency".
func GetSimplePath(path models.TransactionPath) []string {
	var simplePath []string

	if len(path) == 0 {
		return simplePath
	}

	for i, trade := range path {
		simplePath = append(simplePath, fmt.Sprintf("%s:%s", trade.From.Exchange, trade.From.Currency))

		// if it's the last trade in the path, also add the 'To' asset
		if i == len(path)-1 {
			simplePath = append(simplePath, fmt.Sprintf("%s:%s", trade.To.Exchange, trade.To.Currency))
		}
	}

	return simplePath
}
