package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	Unquoted = iota
	DoubleQuote
	SingleQuote

	LineComment = iota
	BlockComment
)

type Node interface {
	_node()
	Pos() ranges.Range
	SetPos(start, end lexer.Position)
}

type Value interface {
	Node
	value()
}

type baseNode struct {
	Range ranges.Range
}

type Document struct {
	baseNode
	Variables map[string]Value
	Body      Value
	Comments  []*Comment
}

type Bool struct {
	baseNode
	Value bool
}

type StringGroup struct {
	baseNode
	Values []Value
}

type String struct {
	baseNode
	Value string
	Quote rune // 0 if unquoted or " or ' rune
}

type Number struct {
	baseNode
	Source string
	Value  float64
}

type List struct {
	baseNode
	Inline bool // Uses brackets
	Items  []Value
}

type Object struct {
	baseNode
	Fields []*Field
	Inline bool // Uses braces
}

type Field struct {
	baseNode
	Key   string
	Value Value
}

type VarRef struct {
	baseNode
	Name   string
	Braces bool // Name wrapped in braces
}

type Comment struct {
	baseNode
	Block  bool
	Source string
}

type Class struct {
	baseNode
	Name string
}

type ArrowRef struct {
	baseNode
	Var *VarRef
}

type None struct{ baseNode }
