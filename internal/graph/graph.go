package graph

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// CycleError represents a circular dependency error.
type CycleError[T comparable] struct {
	Cycle []T
}

func (e *CycleError[T]) Error() string {
	var b strings.Builder
	b.WriteString("circular dependency detected: ")
	for i, node := range e.Cycle {
		if i > 0 {
			b.WriteString(" -> ")
		}
		fmt.Fprintf(&b, "%v", node)
	}
	return b.String()
}

// Graph represents a directed acyclic graph (DAG).
type Graph[T comparable] struct {
	edges    [][2]T
	vertices []T
}

// New creates a new directed graph.
func New[T comparable]() *Graph[T] {
	return &Graph[T]{}
}

func NewWithCapacity[T comparable](capacity int) *Graph[T] {
	return &Graph[T]{edges: make([][2]T, 0, capacity)}
}

// AddEdge adds a directed edge from 'from' to 'to' (from -> to).
// In a dependency graph, this typically means 'to' depends on 'from'.
func (g *Graph[T]) AddEdge(from, to T) {
	g.edges = append(g.edges, [2]T{from, to})
}

func (g *Graph[T]) AddVertex(vertex T) {
	g.vertices = append(g.vertices, vertex)
}

// Port of the JavaScript implementation of ProCode's Algorithm
// See: https://github.com/ProCode-Software/TopoBench
func (g *Graph[T]) Toposort() (sorted []T, err error) {
	// 1. Create the initial dependency map
	deps := make(map[T][]T, len(g.vertices)+len(g.edges)/4)
	for _, vertex := range g.vertices {
		deps[vertex] = nil
	}
	for _, edge := range g.edges {
		dependency, dependent := edge[0], edge[1]
		// 2. For every `a -> b`, append `a` to `depMap[b]`
		deps[dependent] = append(deps[dependent], dependency)
		
		// This is for #1
		if _, ok := deps[dependency]; !ok {
			deps[dependency] = nil
		}
	}
	// 3. For every { a: [b, c] }, append the direct dependencies of b and c and
	// the direct deps of those
	for v := range deps {
		for i := 0; i < len(deps[v]); i++ {
			dep := deps[v][i]
			if dep == v {
				// Cycle
				return nil, nil // TODO
			}
			deps[v] = append(deps[v], deps[dep]...)
		}
	}
	// 4. Whichever vertice has the least dependencies goes first
	return slices.SortedFunc(maps.Keys(deps), func(a, b T) int {
		return len(deps[a]) - len(deps[b])
	}), nil
}
