package engine

import (
	"log/slog"

	"github.com/life00/arbitrage-inspector/internal/models"
)

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

// FindBalances calculates the conversion value of the reference asset into all other assets
// using a breadth first search (BFS) to find the most direct market path.
func FindBalances(pairsPtr *models.Pairs, assetsPtr *models.AssetIndexes, referenceAsset models.AssetBalance) models.AssetBalances {
	assetBalances := make(models.AssetBalances)

	// initialize the source asset balance
	startAssetKey := referenceAsset.Asset
	assetBalances[startAssetKey] = referenceAsset

	// create an adjacency list for fast lookup
	adjacencyList := make(map[models.AssetKey][]models.Pair)
	for _, pair := range *pairsPtr {
		// We use the AssetKey from the AssetIndex to ensure we map correctly
		fromKey := pair.From.Asset
		adjacencyList[fromKey] = append(adjacencyList[fromKey], pair)
	}

	// BFS queue
	queue := []models.AssetKey{startAssetKey}

	// visited map
	visited := make(map[models.AssetKey]bool)
	visited[startAssetKey] = true

	// run the algorithm
	for len(queue) > 0 {
		currentKey := queue[0]
		queue = queue[1:]

		currentBalance := assetBalances[currentKey].Balance

		// check all possible trades
		for _, pair := range adjacencyList[currentKey] {
			neighborKey := pair.To.Asset

			if !visited[neighborKey] {
				visited[neighborKey] = true

				// calculate the new balance: CurrentBalance * Price
				newBalance, _ := currentBalance.Mul(pair.Weight)

				assetBalances[neighborKey] = models.AssetBalance{
					Asset:   neighborKey,
					Balance: newBalance,
				}

				queue = append(queue, neighborKey)
			}
		}
	}

	return assetBalances
}
