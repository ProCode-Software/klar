// Package ast provides token types for the Klar abstract syntax tree (AST).
package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

// ASTItem is implemented by any Klar AST token that has a Kind method returning the
// type of the token.
type ASTItem interface {
	Kind() string
}

// Statement is an [ASTItem] that is a type of EOS-terminated statement in Klar.
type Statement interface {
	ASTItem
	Statement()
}

// Expression is an [ASTItem] that is a type of expression in Klar.
type Expression interface {
	ASTItem
	Expression()
}

// A Program is a parsed Klar file. Body contains the parsed statements in the program,
// and all comments are moved to Comments.
type Program struct {
	Body     []ASTItem
	Comments []Comment `json:"Comments,omitempty"`
}

// An ExpressionStatement is an expression used as a statement. These kind of expressions
// are considered as unused values.
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

// The type of a Comment is one of these.
type CommentType int

const (
	LineComment CommentType = iota
	BlockComment
)

type BinaryExpression struct {
	Left, Right ASTItem
	Operator    lexer.TokenType
}

type UnaryExpression struct {
	Operator lexer.TokenType
	Right    ASTItem
}

// A StringEscape is an escape sequence inside a [StringLiteral].
type StringEscape struct {
	Type  lexer.StringEscapeType
	Index lexer.Position
	Value StringEscapeValue
}

// StringEscapeValue is the value of an escape sequence in a [StringLiteral].
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

type Symbol struct {
	Identifier string
}

type VariableDeclaration struct {
	Identifier   string
	Value        Expression
	Constant     bool // Constant if Identifier is capitalized
	ExplicitType SimpleType
}

type AssignmentStatement struct {
	Assignee Expression
	Operator lexer.TokenType
	Value    Expression
}

// ReservedKeywords is the set of keywords that cannot be used as variables
var ReservedKeywords = map[lexer.TokenType]bool{
	lexer.Import:  true,
	lexer.Func:    true,
	lexer.When:    true,
	lexer.Return:  true,
	lexer.For:     true,
	lexer.Next:    true,
	lexer.Type:    true,
	lexer.Boolean: true,
	lexer.Nil:     true,
}

type Type interface {
	Type()
}
type SimpleType interface {
	Type
	SimpleType()
}

// A PrimitiveType is a Klar-builtin type
type PrimitiveType struct{ Primitive PrimitiveTypeName }

// A TypeAlias is a non-primitive type name
type TypeAlias struct{ Identifier string }

// An OptionalType is a type marked with the suffix '?'. In Klar, this indicates
// that a type could be nil.
type OptionalType struct{ Value Type }
type ListType struct{ Value Type }
type RestType struct{ Value Type }
type TupleType struct{ Values []Type }
type InterfaceType struct{ Fields []TypePair }

type FunctionType struct {
	Parameters []Type
	ReturnType Type
}
type GenericType struct {
	Name       Type
	Parameters []Type
}
type TypePair struct {
	Key   string
	Value Type
}
type UnionType struct {
	Left, Right Type
	Operator    lexer.TokenType
}
type TypeAnnotation struct {
	Variable Symbol
	Type     SimpleType
}

// Primitives
type PrimitiveTypeName int

const (
	PrimitiveString PrimitiveTypeName = iota
	PrimitiveInt
	PrimitiveFloat
	PrimitiveBool
	PrimitiveMap
	PrimitiveNothing
	PrimitiveResult
	PrimitiveError
)

var PrimitiveTypeMap = map[string]PrimitiveTypeName{
	"String":  PrimitiveString,
	"Int":     PrimitiveInt,
	"Float":   PrimitiveFloat,
	"Bool":    PrimitiveBool,
	"Map":     PrimitiveMap,
	"Nothing": PrimitiveNothing,
	"Result":  PrimitiveResult,
	"Error":   PrimitiveError,
}

// Examples:
//
//	import klar.http
//	import klar.http.*
//	import klar.regex.{*}
//	import klar.regex.{type *}
//	import klar.regex.{type RegEx}
//	import fetch: klar.http.requests.{get}
type ImportStatement struct {
	Module             string
	Alias              string // Only if there are no unqualified imports
	UnqualifiedImports []UnqualifiedImport
}

type UnqualifiedImport struct {
	TypeImport bool
	Wildcard   bool
	Identifier string
	Alias      string
}
