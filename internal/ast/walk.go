package ast

import "iter"

// A Visitor is a function that is called for each node in the AST.
type Visitor func(c *Cursor) StopCode

// StopCode is a code that is returned by a visitor to control the walk.
type StopCode int

const (
	ContinueWalk StopCode = iota
	StopWalk              // Stop the walk
	SkipChildren          // Skip the children of the current node
	SkipParent            // Skip the rest of the parent node
	SkipList              // Skip the rest of the items in the list
)

// A Cursor points to a node in the AST.
type Cursor struct {
	parent      *Cursor
	node        Node
	depth       int
	fieldI      int // Index of the current field in parent struct
	*cursorList     // If the cursor points to a list item
}

type cursorList struct {
	index      int
	prev, next *Cursor
}

// Node returns the node c points to.
func (c *Cursor) Node() Node { return c.node }

// Parent returns the parent cursor.
func (c *Cursor) Parent() *Cursor { return c.parent }

// Depth returns the depth of the current cursor since the walk started.
func (c *Cursor) Depth() int { return c.depth }

// IsList reports whether the current cursor points to an item in a slice.
func (c *Cursor) IsList() bool { return c.cursorList != nil }

// FieldIndex returns the index of the field containing c.Node() in its parent.
func (c *Cursor) FieldIndex() int { return c.fieldI }

// Contains reports whether c2 is a descendant of c.
func (c *Cursor) Contains(c2 *Cursor) bool { return false }

// Index returns the index of the current cursor in the list.
func (l *cursorList) Index() int { return l.index }

// Next returns a [Cursor] to the next item in the list.
func (l *cursorList) Next() *Cursor { return l.next }

// Prev returns a [Cursor] to the previous item in the list.
func (l *cursorList) Prev() *Cursor { return l.prev }

// Items returns an iterator over the items in the list.
func (l *cursorList) Items() iter.Seq2[int, *Cursor] { return nil }

// TODO: delete, replace, insert{Before,After}

type walkItem interface {
	fieldIndex() int
	isList() bool
	items() iter.Seq2[int, Node]
}

type walkSlice[T Node] struct {
	index int
	nodes []T
}

func (w walkSlice[T]) isList() bool    { return true }
func (w walkSlice[T]) fieldIndex() int { return w.index }
func (w walkSlice[T]) items() iter.Seq2[int, Node] {
	return func(yield func(int, Node) bool) {
		for i, item := range w.nodes {
			if !yield(i, item) {
				return
			}
		}
	}
}

type walkNode struct {
	index int
	node  Node
}

func (w walkNode) isList() bool    { return false }
func (w walkNode) fieldIndex() int { return w.index }
func (w walkNode) items() iter.Seq2[int, Node] {
	return func(yield func(int, Node) bool) { yield(w.index, w.node) }
}

// TODO
// indices is a slice of field indices for the fields in parent.
func walkFields(v Visitor, parent Node, c *Cursor, items ...walkItem) StopCode {
	// Walk the parent
	var depth, fieldI int
	if c != nil {
		depth = c.depth + 1
		fieldI = c.fieldI
	}
	pc := &Cursor{
		parent: c,
		node:   parent,
		depth:  depth,
		fieldI: fieldI,
	}
	switch v(pc) {
	case StopWalk:
		return StopWalk
	case SkipChildren:
		return ContinueWalk
	case SkipParent:
		return SkipChildren
	case SkipList:
		return SkipList
	case ContinueWalk:
	}

	// Fields of the node (children)
childrenLoop:
	for _, child := range items {
		if child.isList() {
			// Slice of Nodes
		listLoop:
			for i, li := range child.items() {
				// Parent cursor of the list
				c := &Cursor{
					parent: pc,
					fieldI: child.fieldIndex(),
					depth:  pc.depth + 1,
				}
				_, _ = i, c
				switch li.Walk(v, pc) {
				case StopWalk:
					return StopWalk
				case SkipChildren:
					continue listLoop
				case SkipList:
					break listLoop
				case SkipParent:
					return SkipChildren
				case ContinueWalk:
				}
			}
		} else {
			// Single Node
			switch child.(walkNode).node.Walk(v, pc) {
			case StopWalk:
				return StopWalk
			case SkipChildren, SkipList:
				break childrenLoop
			case SkipParent:
				return SkipChildren
			case ContinueWalk:
			}
		}
	}
	return 0
}
