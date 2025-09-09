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

func FindArbitrage(pairsPtr *models.Pairs, assetsPtr *models.Assets, indexPtr *models.Index, sourceAsset models.AssetKey) models.TransactionPath {
	slog.Info("finding arbitrage path...")

	assets := *assetsPtr
	source := assets[sourceAsset].Index

	// prepare the inputs
	slog.Debug("creating graph data...")
	edges := getEdges(pairsPtr)
	vertices := getVertices(assetsPtr)
	graph := newGraph(edges, vertices)

	// run the algorithm
	cyclePath := graph.findArbitrageCycle(source)

	if cyclePath == nil {
		return nil
	}

	// translate the output if a cycle is found
	return translatePath(cyclePath, indexPtr)
}
