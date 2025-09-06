package arbitrage

import (
	"log/slog"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func CreateAssetPairs(exchangesPtr *models.Exchanges, capital decimal.Decimal) (models.Assets, models.Index, models.Pairs) {
	if exchangesPtr == nil || len(*exchangesPtr) == 0 {
		slog.Warn("exchanges data is empty")
		return nil, nil, nil
	}
	slog.Info("creating asset pairs...")
	assets, index := createAssetIndex(exchangesPtr)
	// 2.
	// concurrently create within exchange asset pairs
	// 3.
	// concurrently create inter-exchange asset pairs
	return assets, index, nil
}
