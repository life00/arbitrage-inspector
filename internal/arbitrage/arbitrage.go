package arbitrage

import (
	"log/slog"
	"maps"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func CreateAssetPairs(exchangesPtr *models.Exchanges, capital decimal.Decimal) (models.Pairs, models.Assets, models.Index) {
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

func FindArbitrage(pairs *models.Pairs, assets *models.Assets, index *models.Index, sourceAsset models.AssetKey) models.TransactionPath {
	// convert input into vertices and edges data format
	// create a graph data type using vertices and edges
	// run bellman-ford negative cycle algorithm
	// translate index transaction into TransactionPath type
	return nil
}
