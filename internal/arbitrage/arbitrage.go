package arbitrage

import (
	"log/slog"
	"maps"

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

	pairs := make(models.Pairs)

	intraExchangePairs := createIntraExchangePairs(exchangesPtr, &assets)
	maps.Copy(pairs, intraExchangePairs)

	interExchangePairs := createInterExchangePairs(exchangesPtr, &assets, capital)
	maps.Copy(pairs, interExchangePairs)

	return assets, index, pairs
}
