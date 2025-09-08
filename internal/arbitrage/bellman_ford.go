package arbitrage

import (
	"log/slog"
	"math"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func getEdges(pairsPtr *models.Pairs) []*Edge {
	pairs := *pairsPtr
	var edges []*Edge

	for _, pair := range pairs {
		logWeight, _ := pair.Weight.Log()
		negLogWeight := logWeight.Neg()
		weight, _ := negLogWeight.Float64()

		edge := newEdge(pair.From.Index, pair.To.Index, weight)
		edges = append(edges, edge)
	}
	return edges
}

func getVertices(assetsPtr *models.Assets) []uint {
	length := len(*assetsPtr)
	var vertices []uint

	for i := range length {
		vertices = append(vertices, uint(i))
	}
	return vertices
}

// Graph represents a graph consisting of edges and vertices
type Graph struct {
	edges    []*Edge
	vertices []uint
}

// Edge represents a weighted connection between two nodes
type Edge struct {
	From, To uint
	Weight   float64
}

// newEdge returns a pointer to a new Edge
func newEdge(from, to uint, weight float64) *Edge {
	return &Edge{From: from, To: to, Weight: weight}
}

// newGraph returns a graph consisting of edges and vertices
func newGraph(edges []*Edge, vertices []uint) *Graph {
	return &Graph{edges: edges, vertices: vertices}
}

func (g *Graph) findArbitrageCycle(source uint) []uint {
	slog.Debug("running algorithm...")
	predecessors, distances := g.bellmanFord(source)
	return g.findNegativeWeightCycle(predecessors, distances, source)
}

// bellmanFord determines the shortest path and returns the predecessors and distances
func (g *Graph) bellmanFord(source uint) ([]uint, []float64) {
	size := len(g.vertices)
	distances := make([]float64, size)
	predecessors := make([]uint, size)
	for _, v := range g.vertices {
		distances[v] = math.MaxFloat64
	}
	distances[source] = 0

	for i, changes := 0, 0; i < size-1; i, changes = i+1, 0 {
		for _, edge := range g.edges {
			if newDist := distances[edge.From] + edge.Weight; newDist < distances[edge.To] {
				distances[edge.To] = newDist
				predecessors[edge.To] = edge.From
				changes++
			}
		}
		if changes == 0 {
			break
		}
	}
	return predecessors, distances
}

// findNegativeWeightCycle finds a negative weight cycle from predecessors and a source
func (g *Graph) findNegativeWeightCycle(predecessors []uint, distances []float64, source uint) []uint {
	for _, edge := range g.edges {
		if distances[edge.From]+edge.Weight < distances[edge.To] {
			return reconstructPath(predecessors, source)
		}
	}
	return nil
}

func reconstructPath(predecessors []uint, source uint) []uint {
	size := len(predecessors)
	path := make([]uint, size)
	path[0] = source

	exists := make([]bool, size)
	exists[source] = true

	indices := make([]uint, size)

	var index, next uint
	for index, next = 1, source; ; index++ {
		next = predecessors[next]
		path[index] = next
		if exists[next] {
			return path[indices[next] : index+1]
		}
		indices[next] = index
		exists[next] = true
	}
}

func translatePath(cyclePath []uint, indexPtr *models.Index) models.TransactionPath {
	slog.Debug("translating path...")
	index := *indexPtr
	var transactionPath models.TransactionPath
	length := len(cyclePath)

	if length < 2 {
		return transactionPath
	}

	for i := range length - 1 {
		fromAsset := index[cyclePath[i]]
		toAsset := index[cyclePath[i+1]]

		transactionPath = append(transactionPath,
			models.PairKey{
				From: fromAsset,
				To:   toAsset,
			},
		)
	}

	return transactionPath
}
