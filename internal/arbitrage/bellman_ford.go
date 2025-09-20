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

func getVertices(assetsPtr *models.AssetIndexes) []uint {
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

// NOTE: restructure the function to also share the output of bellmanFord()
// or use bellmanFord() inside of arbitrage.go.
func (g *Graph) findArbitrageCycle(source uint) []uint {
	slog.Debug("running algorithm...")
	// NOTE: run the bellmanFord() algorithm on multi-source node
	predecessors, distances := g.bellmanFord(source)
	return g.findNegativeWeightCycle(predecessors, distances)
	// NOTE: check if any of the source assets are in the cycle
	// if yes then return the cycle as the full transaction path
	// if not then find the shortest path from any of the source assets (multi-source node, based on output
	// from previous bellmanFord() run) to the shortest (cheapest) starting node in the arbitrage cycle (with the most negative weight?)
	// then rerun bellmanFord() with source as the starting node in the arbitrage cycle, to find the shortest (cheapest)
	// path back to any of the source assets (multi-source node)
	// find the product of overall transaction path, and if it is still profitable then return it, otherwise return nothing
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
			newDist := distances[edge.From] + edge.Weight
			if newDist < distances[edge.To] {
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

// findNegativeWeightCycle finds a negative weight cycle from the results of Bellman-Ford
func (g *Graph) findNegativeWeightCycle(predecessors []uint, distances []float64) []uint {
	for _, edge := range g.edges {
		// if this condition is met after the main Bellman-Ford loops, a negative cycle exists
		if distances[edge.From]+edge.Weight < distances[edge.To] {
			// The node edge.To is part of, or reachable from, the negative cycle.
			// To find a node that is *guaranteed* to be on the cycle, we can
			// trace back len(vertices) times. This moves us from any "handle" path
			// onto a node within the cycle itself.
			cycleNode := edge.To
			for range len(g.vertices) {
				cycleNode = predecessors[cycleNode]
			}

			// now, we can reconstruct the path starting from a node we know is on the cycle
			return reconstructPath(predecessors, cycleNode)
		}
	}
	return nil
}

func reconstructPath(predecessors []uint, startNode uint) []uint {
	size := len(predecessors)
	path := make([]uint, 0, size)

	// start with the node known to be in the cycle
	currentNode := startNode

	// use a map to detect when we've completed the cycle
	visited := make(map[uint]bool)

	// trace back predecessors until we find a node we've already seen
	// this marks the beginning and end of the cycle
	for !visited[currentNode] {
		visited[currentNode] = true
		path = append(path, currentNode)
		currentNode = predecessors[currentNode]
	}

	// the currentNode is now the first node of the cycle that we encountered again
	// we need to find where this node appeared in our path to isolate the cycle
	cycleStartIndex := -1
	for i, node := range path {
		if node == currentNode {
			cycleStartIndex = i
			break
		}
	}

	// slice the path to get only the cycle
	cycle := path[cycleStartIndex:]

	// add the start node again to close the loop for translation
	cycle = append(cycle, currentNode)

	// reverse the slice to get the correct path order (e.g., a -> b -> c -> a)
	for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
		cycle[i], cycle[j] = cycle[j], cycle[i]
	}

	return cycle
}

func translatePath(cyclePath []uint, indexPtr *models.Index) models.TransactionPath {
	slog.Debug("translating path...")
	index := *indexPtr
	var transactionPath models.TransactionPath
	length := len(cyclePath)

	if length < 2 {
		return transactionPath
	}

	// loop to length-1 because the last element closes the cycle with the first
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
