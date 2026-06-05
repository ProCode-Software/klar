package graph

import (
	"fmt"
	"strings"
)

// CycleError represents a circular dependency error.
type CycleError[T comparable] struct {
	Cycle []T
}

func (e *CycleError[T]) Error() string {
	var sb strings.Builder
	sb.WriteString("circular dependency detected: ")
	for i, node := range e.Cycle {
		if i > 0 {
			sb.WriteString(" -> ")
		}
		fmt.Fprintf(&sb, "%v", node)
	}
	return sb.String()
}

// Graph represents a directed acyclic graph.
type Graph[T comparable] struct {
	edges map[T][]T
	nodes map[T]struct{}
}

// New creates a new directed graph.
func New[T comparable]() *Graph[T] {
	return &Graph[T]{
		edges: make(map[T][]T),
		nodes: make(map[T]struct{}),
	}
}

// AddNode adds a node to the graph without any edges.
func (g *Graph[T]) AddNode(node T) {
	g.nodes[node] = struct{}{}
}

// AddEdge adds a directed edge from 'from' to 'to'.
// In a dependency graph, this typically means 'from' depends on 'to'.
func (g *Graph[T]) AddEdge(from, to T) {
	g.nodes[from] = struct{}{}
	g.nodes[to] = struct{}{}
	g.edges[from] = append(g.edges[from], to)
}

// TopoSort performs a topological sort on the graph using depth-first search.
// If 'from' depends on 'to' (edge from -> to), then 'to' will appear before 'from' in the result.
// If a cycle is detected, it returns a CycleError containing the path of the cycle.
func (g *Graph[T]) TopoSort() ([]T, error) {
	var (
		result  []T
		visited = make(map[T]bool)
		temp    = make(map[T]bool)
		path    []T
	)

	var visit func(node T) error
	visit = func(node T) error {
		if temp[node] {
			// Cycle detected
			// Extract the cycle from the path
			var cycle []T
			inCycle := false
			for _, p := range path {
				if p == node {
					inCycle = true
				}
				if inCycle {
					cycle = append(cycle, p)
				}
			}
			cycle = append(cycle, node)
			return &CycleError[T]{Cycle: cycle}
		}
		if visited[node] {
			return nil
		}

		temp[node] = true
		path = append(path, node)

		for _, dep := range g.edges[node] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		temp[node] = false
		path = path[:len(path)-1]
		visited[node] = true
		result = append(result, node)
		return nil
	}

	for node := range g.nodes {
		if !visited[node] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}
