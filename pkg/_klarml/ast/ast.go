package ast

import "github.com/ProCode-Software/klar/internal/ranges"

type (
	Range      = ranges.Range
	QuoteStyle int
	baseNode   struct{ Range Range }
)

const (
	SingleQuote QuoteStyle = iota
	DoubleQuote
	Unquoted
)

// Document
type Document struct {
	baseNode
	Variables []*VarDecl
	Comments  []*Comment
	Body      Value
}

type Comment struct {
	baseNode
	Block   bool
	Content string
}
type VarDecl struct {
	baseNode
	Name  string
	Value Value
}

// Values
type Object struct {
	baseNode
	Properties []*Property
}
type Array struct {
	baseNode
	Items  []Value
	Inline bool
}
type Property struct {
	baseNode
	Key   string
	Value Value
}
type StringLiteral struct {
	baseNode
	Content    string
	QuoteStyle QuoteStyle
}
type NumericLiteral struct {
	baseNode
	Value float64
}
type BoolLiteral struct {
	baseNode
	Value bool
}
type Namespace struct {
	baseNode
	Name string
}
type VarRef struct {
	baseNode
	Identifier string
	Braced     bool
}
type ConcatString struct {
	Values []Value
}
type Bad struct {
	baseNode
}
