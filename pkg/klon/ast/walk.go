package ast

type VisitFunc func(n Node, depth int) StopCode

// StopCode copied from https://github.com/ProCode-Software/klar/blob/main/internal/ast/walk.go

// StopCode is a code that is returned by a visitor to control the walk.
type StopCode int

const (
	ContinueWalk StopCode = iota
	StopWalk              // Stop the walk
	SkipChildren          // Skip the children of the current node
	SkipParent            // Skip the rest of the parent node
	SkipList              // Skip the rest of the items in the list
)

// Walk performs a depth-first search from node.
func Walk(node Node, visit VisitFunc) {
}
