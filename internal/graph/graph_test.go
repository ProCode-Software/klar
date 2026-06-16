package graph

import (
	"slices"
	"testing"
)

func input(graph [][2]int) *Graph[int] {
	g := New[int]()
	for _, e := range graph {
		g.AddEdge(e[0], e[1])
	}
	return g
}

func TestToposort(t *testing.T) {
	inputs := []*Graph[int]{
		input(nil),
	}
	for _, g := range inputs {
		sorted, err := g.Toposort()
		if err != nil {
			t.Errorf("expected no cycle")
		}

		// Validate the order
		exists := make(map[int]bool)
		for _, v := range sorted {
			if exists[v] {
				t.Errorf("toposort result %v is out of order: %v is used before it is ready", sorted, v)
			}
			exists[v] = true
		}
	}
}

func TestToposortCycles(t *testing.T) {
	inputs := []struct {
		graph  *Graph[int]
		cycles [][]int
	}{
		{func() *Graph[int] {
			g := New[int]()
			g.AddEdge(1, 3)
			g.AddEdge(2, 3)
			g.AddEdge(2, 5)
			g.AddEdge(5, 3)
			g.AddEdge(7, 5)
			g.AddEdge(6, 7)
			g.AddEdge(4, 6)
			g.AddEdge(3, 4)
			g.AddEdge(3, 6)
			return g
		}(), [][]int{{3, 5, 7, 6, 4, 3}}},
	}
	for _, g := range inputs {
		sortedForSomeReason, err := g.graph.Toposort()
		if err == nil {
			t.Errorf("expected a cycle, got %v", sortedForSomeReason)
		}
		cycleErr := err.(*CycleError[int])
		if !slices.ContainsFunc(g.cycles, func(option []int) bool {
			return slices.Equal(option, cycleErr.Cycle)
		}) {
			t.Errorf("expected cycle to be one of %v, got %v", g.cycles, cycleErr.Cycle)
		}
	}
}
