package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Node interface {
	Pos() ranges.Range
	SetPos(start, end lexer.Position)
}

type Value = Node

// BaseNode
// ============

type BaseNode struct{ Range ranges.Range }

func (b *BaseNode) Pos() ranges.Range { return b.Range }
func (b *BaseNode) SetPos(start, end lexer.Position) {
	b.Range = ranges.Range{Start: start, End: end}
}

type Document struct {
	BaseNode
	Variables map[string]Value
	Body      Value
	Comments  []*Comment
}

type Boolean struct {
	BaseNode
	Value bool
}

type StringGroup struct {
	BaseNode
	Values []Value
}

type String struct {
	BaseNode
	Raw   string   // Input string
	Value []string // Escapes evaluated, variables as segments
	Wrap  bool     // If '>' was before quote
	Quote rune     // 0 if unquoted or " or ' rune
}

type Number struct {
	BaseNode
	Source string
	Value  float64
}

type List struct {
	BaseNode
	Inline bool // Uses brackets
	Items  []Value
}

type Object struct {
	BaseNode
	Fields []*Field
	Inline bool // Uses braces
}

type Field struct {
	BaseNode
	Key     Value
	KeyPath *[]Value
	Value   Value

	Arrow *ArrowRef
}

type VarRef struct {
	BaseNode
	Name   string
	Braces bool // Name wrapped in braces
}

type Comment struct {
	BaseNode
	Block  bool
	Source string
}


type Class struct {
	BaseNode
	Name string
}

type ArrowRef struct {
	BaseNode
	Var *VarRef
}

type Bad struct {
	BaseNode
	Value any
}

func (b *Bad) Pos() ranges.Range {
	if !b.BaseNode.Range.IsZero() {
		return b.BaseNode.Range
	}
	if node, ok := b.Value.(Node); ok {
		return node.Pos()
	}
	return ranges.Range{}
}

type None struct{ BaseNode }
