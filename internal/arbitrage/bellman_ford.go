package arbitrage

import (
	"math"
)

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

// NewEdge returns a pointer to a new Edge
func NewEdge(from, to uint, weight float64) *Edge {
	return &Edge{From: from, To: to, Weight: weight}
}

// NewGraph returns a graph consisting of edges and vertices
func NewGraph(edges []*Edge, vertices []uint) *Graph {
	return &Graph{edges: edges, vertices: vertices}
}

// FindArbitrageCycle finds a negative-weight cycle starting from and returning to the source
func (g *Graph) FindArbitrageCycle(source uint) []uint {
	// run Bellman-Ford to find the shortest paths from the source
	predecessors, distances := g.BellmanFord(source)

	// check for a negative cycle involving the source
	for _, edge := range g.edges {
		if edge.To == source {
			// found an edge returning to the source
			// check if the path from source -> edge.From -> source has a negative weight
			// distances[source] is 0, so we check if distances[edge.From] + edge.Weight < 0
			if distances[edge.From] != math.MaxFloat64 && distances[edge.From]+edge.Weight < 0 {
				// arbitrage opportunity found
				// the cycle is the path from source to edge.from, plus the edge back to source
				path := reconstructPath(predecessors, source, edge.From)
				return append(path, source)
			}
		}
	}

	// no negative cycles detected
	return nil
}

// standard Bellman-Ford algorithm
func (g *Graph) BellmanFord(source uint) ([]uint, []float64) {
	size := len(g.vertices)
	distances := make([]float64, size)
	predecessors := make([]uint, size)
	for _, v := range g.vertices {
		distances[v] = math.MaxFloat64
	}
	distances[source] = 0

	// relax edges |V|-1 times
	for range size - 1 {
		for _, edge := range g.edges {
			if distances[edge.From] != math.MaxFloat64 && distances[edge.From]+edge.Weight < distances[edge.To] {
				distances[edge.To] = distances[edge.From] + edge.Weight
				predecessors[edge.To] = edge.From
			}
		}
	}
	return predecessors, distances
}

// reconstructPath builds the path from a source to a destination using the predecessors array
func reconstructPath(predecessors []uint, source, dest uint) []uint {
	var path []uint
	// trace backwards from destination to source
	for curr := dest; curr != source; curr = predecessors[curr] {
		path = append(path, curr)
	}
	path = append(path, source)

	// reverse the path to get the correct order (source -> dest)
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
