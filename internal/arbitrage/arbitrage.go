package arbitrage

import (
	"log/slog"
	"maps"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

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

func FindArbitrage(pairsPtr *models.Pairs, assetsPtr *models.AssetIndexes, indexPtr *models.Index, sourceAsset models.AssetKey) models.TransactionPath {
	if len(*assetsPtr) == 0 {
		return nil
	}

	slog.Info("finding arbitrage path...")

	assets := *assetsPtr
	// NOTE: use a multi-source node instead of a single asset source
	// A_1, A_2, A_n source assets would all have `0` weight edges to **S** (multi-source node)
	// this allows to find the cheapest transfer path from any of the source assets
	// to the arbitrage cycle if none of the source assets are in the arbitrage cycle.
	// It is probably better to implement most of this inside of pairs.go Pairs data generation
	source := assets[sourceAsset].Index

	// prepare the inputs
	slog.Debug("creating graph data...")
	edges := getEdges(pairsPtr)
	vertices := getVertices(assetsPtr)
	graph := newGraph(edges, vertices)

	// run the algorithm
	// TODO: completely refactor the algorithm, because if none of the source assets are
	// in the arbitrage cycle, it must determine the cheapest path from the source assets (multi-source node)
	// to the arbitrage cycle starting node, and back to the source assets (multi-source node)
	// then it evaluates whether it is still profitable, if yes then it returns the overall path, otherwise no arbitrage is found.
	// Please check all the notes in the FindArbitrage() function
	cyclePath := graph.findArbitrageCycle(source)

	if cyclePath == nil {
		return nil
	}

	// translate the output if a cycle is found
	return translatePath(cyclePath, indexPtr)
}
