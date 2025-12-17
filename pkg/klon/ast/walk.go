package ast

import (
	"errors"
)

type VisitFunc func(n Node, depth int) error

// Special errors that can be returned by [VisitFunc] to control a [Walk] operation.
var (
	StopWalk       = errors.New("stop the walk")
	SkipCollection = errors.New("skip the rest of this collection")
	SkipNode       = errors.New("skip the rest of this node")
)

// Walk performs a breadth-first search from node, until visit
// returns an error when the current node is passed.
func Walk(node Node, visit VisitFunc) error {
	return nil
}
