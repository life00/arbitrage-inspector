// Package engine provides functions to run key algorithms on the data.
package engine

import (
	"log/slog"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func FindArbitrage(pairsPtr *models.Pairs, assetsPtr *models.AssetIndexes, indexPtr *models.Index, sourceAssets models.AssetBalances) models.TransactionPath {
	slog.Debug("finding arbitrage path")
	if len(*assetsPtr) == 0 {
		return nil
	}

	// prepare the inputs
	edges := getEdges(pairsPtr)
	vertices := getVertices(assetsPtr)
	graph := newGraph(edges, vertices)
	graph.addSuperSource(sourceAssets, *assetsPtr)

	// TODO: completely refactor the algorithm, because if none of the source assets are
	// in the arbitrage cycle, it must determine the cheapest path from the source assets (multi-source node)
	// to the arbitrage cycle starting node, and back to the source assets (multi-source node)
	// then it evaluates whether it is still profitable, if yes then it returns the overall path, otherwise no arbitrage is found.
	// NOTE: check if any of the source assets are in the cycle
	// if yes then return the cycle as the full transaction path
	// if not then find the shortest path from any of the source assets (multi-source node, based on output
	// from previous bellmanFord() run) to the shortest (cheapest) starting node in the arbitrage cycle (with the most negative weight?)
	// then rerun bellmanFord() with source as the starting node in the arbitrage cycle, to find the shortest (cheapest)
	// path back to any of the source assets (multi-source node)
	// find the product of overall transaction path, and if it is still profitable then return it, otherwise return nothing

	// run the algorithm
	predecessors, distances := graph.bellmanFord(0) // 0 is the super source
	cyclePath := graph.findNegativeWeightCycle(predecessors, distances)

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
