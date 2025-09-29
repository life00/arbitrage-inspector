package transform

import (
	"log/slog"
	"maps"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

// FindAssetBalances() determines the balance amount of all source assets
// assumes that config.SourceAssets is accessible from config.ReferenceAsset
func FindAssetBalances(config models.Config, exchanges *models.Exchanges) map[models.AssetKey]models.AssetBalance {
	return nil
}

func CreateAssetPairs(exchangesPtr *models.Exchanges, capital decimal.Decimal) (models.Pairs, models.AssetIndexes, models.Index) {
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

	return pairs, assets, index
}
