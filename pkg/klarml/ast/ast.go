package ast

const (
	Unquoted = iota
	DoubleQuote
	SingleQuote

	LineComment = iota
	BlockComment
)

type Node interface {
	Pos() int
	End() int
}

type Value interface {
	Node
	value()
}

type baseNode struct {
	StartPos, EndPos int
}

type Document struct {
	baseNode
	Variables []*VarDecl
	Body      Value
	Comments  []*Comment
}

type Bool struct {
	baseNode
	Value bool
}

type StringSeq struct {
	baseNode
	Values []Value
}

type String struct {
	baseNode
	Value string
	Quote int
}

type Number struct {
	baseNode
	Source string
	Value  float64
}

type Array struct {
	baseNode
	Inline bool
	Items  []Value
}

type Object struct {
	baseNode
	Props []*Prop
}

type Prop struct {
	baseNode
	Key   string
	Path  []string
	Value Value
}

type VarDecl struct {
	baseNode
	Name  string
	Value Value
}

type Comment struct {
	baseNode
	Type   int
	Source string
}

type Class struct {
	baseNode
	Name string
}

type Null struct{ baseNode }
