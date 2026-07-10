package graph

import (
	"cmp"
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
	b.WriteString("graph has circular dependencies: ")
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
	// Used for sorting vertices with the same dependency count for deterministic output.
	Compare func(T, T) int
}

// New creates a new directed graph.
func New[T comparable]() *Graph[T] {
	return &Graph[T]{}
}

func NewWithCompare[T comparable](compare func(T, T) int) *Graph[T] {
	return &Graph[T]{Compare: compare}
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

func (g *Graph[T]) AllVertices() []T {
	set := make(map[T]struct{})
	for _, vertex := range g.vertices {
		set[vertex] = struct{}{}
	}
	for _, edge := range g.edges {
		set[edge[0]] = struct{}{}
		set[edge[1]] = struct{}{}
	}
	return slices.Collect(maps.Keys(set))
}

func (g *Graph[T]) Edges() [][2]T { return g.edges }

// Port of the JavaScript implementation of ProCode's Algorithm
// See: https://github.com/ProCode-Software/procosort
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
				return nil, &CycleError[T]{g.trackCycle(v)}
			}
			deps[v] = append(deps[v], deps[dep]...)
		}
	}
	// 4. Whichever vertice has the least dependencies goes first
	_, isString := any(*new(T)).(string)
	return slices.SortedFunc(maps.Keys(deps), func(a, b T) int {
		byDepOrder := len(deps[a]) - len(deps[b])
		switch {
		case byDepOrder != 0:
			return byDepOrder
		case g.Compare != nil:
			return g.Compare(a, b)
		case isString:
			// Sort by name if items are strings, for deterministic output
			return cmp.Compare(any(a).(string), any(b).(string))
		}
		return 0
	}), nil
}

func (g *Graph[T]) ToposortGrouped() (groups [][]T, err error) {
	return nil, nil
}

func (g *Graph[T]) trackCycle(v T) []T {
	/* The item in the cycle is 3 [x=3]
	Look for dependencies of v
	1. (1, 3). 1 has no dependencies, next
	2. (2, 3). 2 has no dependencies, next
	3. (5, 3). Yes [3 refers to 5] [x=5]
	4. (2, 5) and (7, 5) are options.
	5. For (2, 5): 2 has no dependencies, next
	6. For (7, 5): Found (6, 7) [7 refers to 6] [x = 7, then x=6]
	7. Options are (4, 6) and (3, 6)
	8. For (4, 6) [6 refers to 4]: Found (3, 4) [4 refers to 3] [x = 4, then x = 3]
	9. Cycle ends

	So:
		3 refers to 5
		5 refers to 7
		7 refers to 6
		6 refers to 4
		4 refers to 3
	Cycle is [3, 5, 7, 6, 4] */

	findDependencies := func(dependent T) (deps []T) {
		for _, edge := range g.edges {
			if edge[1] == dependent {
				deps = append(deps, edge[0])
			}
		}
		return
	}
	var tryDependency func(T) ([]T, bool)
	tryDependency = func(dep T) (cycle []T, ok bool) {
		deps := findDependencies(dep)
		if len(deps) == 0 {
			return nil, false
		}
		for _, dep := range deps {
			if dep == v {
				cycle = append(cycle, v)
				return cycle, true
			}
			if cycle, ok := tryDependency(dep); ok {
				cycle = append(cycle, dep)
				return cycle, true
			}
		}
		return nil, false
	}

	cycle, ok := tryDependency(v)
	if !ok {
		panic(fmt.Sprintf("impossible: no dependencies for vertex %v", v))
	}
	cycle = append(cycle, v)
	slices.Reverse(cycle)
	return cycle
}
