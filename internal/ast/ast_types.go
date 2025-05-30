// Package ast provides token types for the Klar abstract syntax tree (AST).
package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

// All AST tokens implement the Node interface.
type Node interface {
	Kind() string
}

// All EOS-terminated statement AST tokens implement the Statement interface.
type Statement interface {
	Node
	Statement()
}

// All expression AST tokens implement the Expression interface.
type Expression interface {
	Node
	Expression()
}

type BadExpression struct{ Value Node }

// A Program is a parsed Klar file. Body contains the parsed statements in the program,
// and all comments are moved to Comments.
type Program struct {
	Body     []Statement
	Comments []Comment
}

// An ExpressionStatement is an expression used as a statement.
type ExpressionStatement struct {
	Expression Expression
}

type StringLiteral struct {
	QuoteStyle rune
	Content    string
	Escapes    map[lexer.Position]StringEscape
}

type IntegerLiteral struct {
	Format int
	Value  int
}

type BooleanLiteral struct{ Value bool }

type NilLiteral struct{}

type FloatLiteral struct{ Value float64 }

type Comment struct {
	Begin, End lexer.Position
	Type       CommentType
	Value      string
}

// lexer.LineComment or lexer.BlockComment
type CommentType = lexer.TokenType

type BinaryExpression struct {
	Left, Right Node
	Operator    lexer.TokenType
}

type UnaryExpression struct {
	Operator lexer.TokenType
	Right    Node
}

// A StringEscape is an escape sequence inside a [StringLiteral].
type StringEscape interface {
	StringEscape()
}

type BadEscape struct{ Source string }
type CharacterEscape struct{ Character rune }
type UnicodeEscape struct{ Hex int32 }
type HexadecimalEscape struct{ Hex int32 }
type StringInterpolation struct{ Expression Node }

type Symbol struct{ Identifier string }

type Discard struct{} // _

type Assignable interface {
	Assignable()
}

// Publicizable is any declaration that allows the `public` modifier.
// Calling the Publicizable will set the Public field to true on the declaration.
type Publicizable interface {
	Publicize()
}

type VariableDeclaration struct {
	Public       bool
	Identifier   string
	Value        Expression
	Constant     bool // Constant if Identifier is capitalized
	ExplicitType SimpleType
}

type AssignmentStatement struct {
	Assignee Assignable
	Operator lexer.TokenType
	Value    Expression
}

type Pair struct {
	Key, Value Expression
}

// ReservedIdent is the set of keywords that cannot be used as variable names.
var ReservedIdent = []lexer.TokenType{
	lexer.Import, lexer.Func, lexer.When, lexer.Return, lexer.For, lexer.Next,
	lexer.Type, lexer.Public, lexer.Boolean, lexer.Nil, lexer.And, lexer.Or, lexer.In,
}

type Type interface {
	Node
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
	Parameters []SimpleType
}
type TypePair struct {
	Key   string
	Value Type
}
type UnionType struct {
	Options []Type
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
	Wildcard           bool
	UnqualifiedImports []UnqualifiedImport
}

type UnqualifiedImport struct {
	TypeImport bool
	Wildcard   bool
	Identifier string
	Alias      string
}

type TypeDeclaration interface {
	Statement
	TypeDeclaration()
}

type StructDeclaration struct {
	Public         bool
	Identifier     string
	InheritedTypes []Type // Type alias or primitive
	Fields         []StructField
}
type StructField struct {
	Identifier string
	Type       Type
	Constant   bool
	Value      Expression
}

type EnumDeclaration struct {
	Public     bool
	Identifier string
	Values     []EnumItem
}
type EnumItem struct {
	Identifier string
	Value      Expression
}

type TypeAliasDeclaration struct {
	Public     bool
	Identifier string
	Type       Type
}

type MapLiteral struct {
	Entries []Pair
}

type TupleLiteral struct {
	Values []Expression
}

type ReturnStatement struct {
	Value Expression // Can be nil
}

// A FunctionDeclaration is a basic Klar function or method declaration.
type FunctionDeclaration struct {
	Public        bool
	Identifier    string
	Struct        Type
	GenericParams []string
	Parameters    []FunctionParam
	ReturnType    SimpleType
	Body          []Statement
	Expression
}

type FunctionParam struct {
	Identifier,
	Label string
	Type    SimpleType
	Default Expression
}

type NextStatement struct{}

type ListLiteral struct {
	Items []Expression
}

type IndexExpression struct {
	Object, Property Node
	Computed         bool // If square bracket [
}

type EnumLiteral struct{ Name string }

type CallParam struct {
	Label string
	Value Expression
}

type CallExpression struct {
	Callee Node
	Args   []CallParam
}

type TypeCastSymbol struct {
	Type SimpleType
}

// An UpdateStatement is a decrement or increment statement. These statements end in
// ++ or --. Unlike other languages such as C, Klar's increment/decrement operators
// are statements rather than expressions.
type UpdateStatement struct {
	Left     Expression
	Operator lexer.TokenType
}

type ForStatement struct {
	Infinite   bool       // or
	Condition  Expression // or
	Variables  []Symbol
	Assignment Expression

	Body []Statement
}

type WhenBlock struct {
	IsExpression bool
	Subjects     []Expression
	Cases        []WhenCase
}

type WhenCase struct {
}

type ParamTuple struct {
	Params []TypePair
}

type LambdaExpression struct {
	Params   []TypePair
	Body     []Statement
	ExprBody Expression
}

type Attribute struct {
	Decorator string
	Args      []CallParam
}

type RestExpression struct {
	Left bool
	Expr Expression
}

type RangeExpression struct {
	Start, End, Step Expression
}

type PipelineExpression struct {
	Steps []Node
}
