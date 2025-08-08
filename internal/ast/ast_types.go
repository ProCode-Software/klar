// Package ast provides token types for the Klar abstract syntax tree (AST).
package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// All AST tokens implement the Node interface.
//
//go:generate stringer -type=PrimitiveTypeName -linecomment
type Node interface {
	GetRange() ranges.Range
	SetPos(start, end lexer.Position)
}

type BaseNode struct {
	Range ranges.Range
}

// All EOS-terminated statement AST tokens implement the Statement interface.
type Statement interface {
	Node
	stmt()
}

// All expression AST tokens implement the Expression interface.
type Expression interface {
	Node
	expr()
}

type BadExpression struct {
	BaseNode
	Value Node
	Token lexer.TokenType
}

// A Program is a parsed Klar file. Body contains the parsed statements in the program,
// and all comments are moved to Comments.
type Program struct {
	BaseNode
	Body     []Statement
	Comments []*Comment
}

// An ExpressionStatement is an expression used as a statement.
type ExpressionStatement struct {
	BaseNode
	Expression Expression
}

type StringLiteral struct {
	BaseNode
	QuoteStyle rune
	Content    string
	Escapes    map[lexer.Position]StringEscape
}

type IntegerLiteral struct {
	BaseNode
	Format int
	Value  int64
	Source string
}

type BooleanLiteral struct {
	BaseNode
	Value bool
}

type NilLiteral struct {
	BaseNode
	Shorthand bool
}

type FloatLiteral struct {
	BaseNode
	Value  float64
	Source string
}

type RegexLiteral struct {
	BaseNode
	Source, Flags string
}

type VersionLiteral struct {
	BaseNode
	Version string
}

type Comment struct {
	BaseNode
	Type  CommentType
	Value string
}

// lexer.LineComment or lexer.BlockComment
type CommentType = lexer.TokenType

type Operator struct {
	Kind     lexer.TokenType
	Position lexer.Position
}

type BinaryExpression struct {
	Left, Right Node
	Operator    Operator
	BaseNode
}

type UnaryExpression struct {
	Operator Operator
	Right    Node
	BaseNode
}

// A StringEscape is an escape sequence inside a [StringLiteral].
type StringEscape interface {
	stringEsc()
}

type (
	BadEscape           struct{ Source string }
	CharacterEscape     struct{ Character rune }
	UnicodeEscape       struct{ Hex int32 }
	HexadecimalEscape   struct{ Hex int32 }
	StringInterpolation struct{ Expression Node }
)

type Symbol struct {
	BaseNode
	Identifier string
}

type Discard struct{ BaseNode } // _

type Assignable interface {
	assignable()
}

type PublicDeclaration struct {
	BaseNode
	Declaration Statement
}

// todo: specify indices that are constant if not symbol
// todo: multiple assignees
type VariableDeclaration struct {
	BaseNode

	Assignee     Expression
	Value        Expression
	Constant     bool // Constant if Identifier is capitalized
	ExplicitType Type
}

type AssignmentStatement struct {
	BaseNode
	Assignee Assignable
	Operator Operator
	Value    Expression
}

type Pair struct {
	BaseNode
	Key, Value Expression
}

// ReservedIdent is the set of keywords that cannot be used as variable names.
var ReservedIdent = []lexer.TokenType{
	lexer.Import, lexer.Func, lexer.When, lexer.Return, lexer.For, lexer.Next,
	lexer.Type, lexer.Public, lexer.Boolean, lexer.Nil, lexer.And, lexer.Or,
	lexer.In, lexer.Break,
}

type Type interface {
	Node
	_type()
}

// A PrimitiveType is a Klar-builtin type
type PrimitiveType struct {
	BaseNode
	Primitive PrimitiveTypeName
}

// A TypeAlias is a non-primitive type name
type TypeAlias struct {
	BaseNode
	Namespace, Identifier string
}

// An OptionalType is a type marked with the suffix '?'. In Klar, this indicates
// that a type could be nil.
type OptionalType struct {
	BaseNode
	Value Type
}
type ListType struct {
	BaseNode
	Value Type
}
type RestType struct {
	BaseNode
	Value Type
}
type TupleType struct {
	BaseNode
	Values []Type
}

type FunctionType struct {
	BaseNode
	Parameters []Type
	ReturnType Type
}
type GenericType struct {
	BaseNode
	Name       Type
	Parameters []Type
}
type TypePair struct {
	Key   string
	Value Type
	BaseNode
}
type UnionType struct {
	BaseNode
	Options []Type
}
type MethodType struct {
	BaseNode
	ReturnType Type
	Parameters []*MethodTypeParam
}
type MethodTypeParam struct {
	Label, Identifier string
	Type              Type
	BaseNode
}
type TypeAnnotation struct {
	BaseNode
	Variable Expression
	Type     Type
}

// Primitives
type PrimitiveTypeName int

const (
	PrimitiveAny     PrimitiveTypeName = iota // Any
	PrimitiveString                           // String
	PrimitiveInt                              // Int
	PrimitiveFloat                            // Float
	PrimitiveBool                             // Bool
	PrimitiveMap                              // Map
	PrimitiveNothing                          // Nothing
	PrimitiveResult                           // Result
	PrimitiveError                            // Error
)

var PrimitiveTypeMap = map[string]PrimitiveTypeName{
	"Any":     PrimitiveAny,
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
	BaseNode
	Module, Alias      *Symbol // Alias is nil if no unqualified imports
	Wildcard           bool
	UnqualifiedImports []*UnqualifiedImport
}

type UnqualifiedImport struct {
	BaseNode
	TypeImport bool
	Wildcard   bool
	Identifier string
	Alias      string
}

type TypeDeclaration interface {
	Statement
	Name() string
	typeDecl()
}

type InterfaceDeclaration struct {
	Identifier     string
	InheritedTypes []Type
	Tag            bool // If empty
	Fields         []*TypePair
	BaseNode
}

type StructDeclaration struct {
	Identifier     string
	InheritedTypes []Type // Type alias or primitive
	Fields         []*StructField
	BaseNode
}
type StructField struct {
	Identifier string
	Type       Type
	Constant   bool
	Value      Expression
	BaseNode
}

type EnumDeclaration struct {
	Identifier string
	Inherited  []Type
	Values     []*EnumItem
	BaseNode
}
type EnumItem struct {
	Identifier string
	Value      Expression
	Parameters []Type
	BaseNode
}

type TypeAliasDeclaration struct {
	Identifier string
	Type       Type
	BaseNode
}

type MapLiteral struct {
	Entries []*Pair
	BaseNode
}

type TupleLiteral struct {
	Values []Expression
	BaseNode
}

type ReturnStatement struct {
	Value Expression // Can be nil
	BaseNode
}

// A FunctionDeclaration is a basic Klar function or method declaration.
type FunctionDeclaration struct {
	Identifier    *Symbol
	Struct        Type
	GenericParams []*Symbol
	Parameters    []*FunctionParam
	ReturnType    Type
	Body          []Statement
	Expression    Expression
	BaseNode
}

type FuncAliasDeclaration struct {
	BaseNode

	Struct     Type
	Identifier *Symbol
	Alias      *Symbol
}

type FunctionParam struct {
	Identifier,
	Label string
	Type    Type
	Default Expression
	BaseNode
}

type NextStatement struct{ BaseNode }

type BreakStatement struct{ BaseNode }

type ListLiteral struct {
	BaseNode
	Items []Expression
}

type IndexExpression struct {
	Object, Property Node
	Computed         bool // If square bracket [
	BaseNode
}

type SliceExpression struct {
	Object        Node
	Index, Length Expression
	BaseNode
}

type EnumLiteral struct {
	BaseNode
	Name string
}

type CallParam struct {
	Label string
	Value Expression
	BaseNode
}

type CallExpression struct {
	Callee Node
	Args   []*CallParam
	BaseNode
}

// An UpdateStatement is a decrement or increment statement. These statements end in
// ++ or --. Unlike other languages such as C, Klar's increment/decrement operators
// are statements rather than expressions.
type UpdateStatement struct {
	Left     Expression
	Operator Operator
	BaseNode
}

// A for statement acts as a foreach (C#), while (C) and loop (Rust) with
// one keyword, similar to Go.
//
//	for { ...infinite loop }
//	for <expr> - while loop
//	for k, v in <expr>
//	for item in <expr>
//	for 5 { ...repeat 5 times } - only if literal, else - for _ in 5
type ForStatement struct {
	BaseNode
	Infinite   bool // or
	Variables  []string
	Expression Expression // When used as while loop or repeat
	Body       []Statement
}

type WhenExpression struct {
	BaseNode
	Subjects []Expression
	Cases    []*WhenCase
}

type WhenCase struct {
	Options  [][]Expression
	Guard    Expression  // <case> when <expr>
	Body     []Statement // -> <expr> | -> {...}
	BodyExpr Expression
	InBraces bool
}

type TypeTuple struct {
	BaseNode
	Params []*TypePair
}

type LambdaExpression struct {
	BaseNode
	Params   []*TypePair
	Body     []Statement
	ExprBody Expression
}

type Attribute struct {
	BaseNode
	Decorator string
	Args      []*CallParam
}

type RestExpression struct {
	BaseNode
	Left bool
	Expr Expression
}

type RangeExpression struct {
	From, To, Step Expression
	BaseNode
}

type PipelineExpression struct {
	Steps []Node
	BaseNode
}

type ParenExpression struct {
	Expr Expression
	BaseNode
}

type ListCastExpression struct {
	BaseNode
	Type Type
	Args []*CallParam
}

// A ForExpression is a [ForStatement] used as an expression. A ForExpression
// can only iterate. It may reduce when the +=, -=, or = operator is used,
// filter when a block is used, or map when -> is used.
//
//	sum := for i in items += i
//	for [variables] in [iterator] [-> | = | += | -=] [value]
//	for [variables] in [iterator] { block... }
type ForExpression struct {
	BaseNode
	Variables []Destructure
	Iterator  Expression
	Value     Expression
	Block     []Statement
}

type ObjectPipeline struct {
	BaseNode
	Object Expression
	Steps  []Node // Assignment or method call
}

// Destructuring
// Values that can be used in VariableDeclaration implement Destructure.
// (a, b) | [a, b] | #{ a, b } | a
type Destructure interface {
	Expression
	Vars() []*Symbol
}

type ListDestructure struct {
	BaseNode
	Tuple  bool
	Values []Destructure
}

// Object or map destructure
type KeyDestructure struct {
	BaseNode
	Values []*KeyDestructureEntry
}

// Entry for [KeyDestructure]
//
//	#{ in: ("John", 14) }    -> #{ in.(name, age) } -> name, age
//	#{ when: true }          -> #{ cond: when }     -> cond
//	#{ data: [#{ key: 0 }] } ->
//		#{ data[{ key }] }        -> key
//		#{ data[{ myKey: key }] } -> myKey
//		#{ data[first] }          -> first
type KeyDestructureEntry struct {
	BaseNode
	Alias  *Symbol // optional
	Object *Symbol // optional
	Index  Destructure
}
