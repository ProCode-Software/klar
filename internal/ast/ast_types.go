// Package ast provides token types for the Klar abstract syntax tree (AST).
package ast

// ASTItem is implemented by any Klar AST token that has a Kind method returning the
// type of the token.
type ASTItem interface {
	Kind() string
}

type Program struct {
	Items []ASTItem
}
type StringLiteral struct {
	QuoteStyle int
	Content    string
}
type IntegerLiteral struct {
	Format int
	Value  int
}
type BooleanLiteral struct {
	Value bool
}
type NilLiteral struct{}
type FloatLiteral struct {
	Value float64
}
type Comment struct {
	Type  int
	Value string
}

func (Program) Kind() string        { return "Program" }
func (StringLiteral) Kind() string  { return "StringLiteral" }
func (FloatLiteral) Kind() string   { return "FloatLiteral" }
func (IntegerLiteral) Kind() string { return "IntegerLiteral" }
func (Comment) Kind() string        { return "IntegerLiteral" }
func (BooleanLiteral) Kind() string { return "BooleanLiteral" }
func (NilLiteral) Kind() string     { return "NilLiteral" }

