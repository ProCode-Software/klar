package graph

import "testing"

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
		if msg, ok := check(sorted); !ok {
			t.Errorf("toposort result is %v is incorrect: %s", sorted, msg)
		}
	}
}

func TestToposortCycles(t *testing.T) {
	inputs := []*Graph[int]{
		input(nil),
	}
	for _, g := range inputs {
		if sortedForSomeReason, err := g.Toposort(); err == nil {
			t.Errorf("expected a cycle, got %v", sortedForSomeReason)
		}
	}
}

func check(result []int) (msg string, ok bool) {
	return "", true
}
