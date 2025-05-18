// Package ast provides token types for the Klar abstract syntax tree (AST).
package ast

import "github.com/ProCode-Software/klar/internal/lexer"

// ASTItem is implemented by any Klar AST token that has a Kind method returning the
// type of the token.
type ASTItem interface {
	Kind() string
}

type Statement interface {
	ASTItem
	Statement()
}

type Expression interface {
	ASTItem
	Expression()
}

type Program struct {
	Body     []ASTItem
	Comments []Comment `json:"Comments,omitempty"`
}

// Expressions used as statements - these would be unused values
type ExpressionStatement struct {
	Expression ASTItem
}

type StringLiteral struct {
	QuoteStyle rune
	Content    string
	Escapes    []StringEscape
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
	Begin, End lexer.Position
	Type       CommentType
	Value      string
}

type CommentType int

const (
	LineComment CommentType = iota
	BlockComment
)

type BinaryExpression struct {
	Left, Right ASTItem
	Operator    lexer.TokenType
}

// String Escapes
type StringEscape struct {
	Type  lexer.StringEscapeType
	Index lexer.Position
	Value StringEscapeValue
}
type StringEscapeValue interface {
	StringEscape()
}
type CharacterEscape struct {
	Character rune
}
type UnicodeEscape struct {
	Hex int32
}
type HexadecimalEscape struct {
	Hex int32
}
type StringInterpolation struct {
	Expression ASTItem
}

// AST items
func (Program) Kind() string             { return "Program" }
func (StringLiteral) Kind() string       { return "StringLiteral" }
func (FloatLiteral) Kind() string        { return "FloatLiteral" }
func (IntegerLiteral) Kind() string      { return "IntegerLiteral" }
func (BooleanLiteral) Kind() string      { return "BooleanLiteral" }
func (NilLiteral) Kind() string          { return "NilLiteral" }
func (ExpressionStatement) Kind() string { return "ExpressionStatement" }
func (BinaryExpression) Kind() string    { return "BinaryExpression" }

// String escapes
func (CharacterEscape) StringEscape()     {}
func (UnicodeEscape) StringEscape()       {}
func (HexadecimalEscape) StringEscape()   {}
func (StringInterpolation) StringEscape() {}

// Expressions
func (BinaryExpression) Expression() {}
func (NilLiteral) Expression()       {}
func (StringLiteral) Expression()    {}
func (IntegerLiteral) Expression()   {}
func (FloatLiteral) Expression()     {}
func (BooleanLiteral) Expression()   {}
