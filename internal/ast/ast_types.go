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
	Source string
	Flags  []rune
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
	Node
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
	Variables    []Destructure
	Value        Expression
	ExplicitType Type
}

type AssignmentStatement struct {
	BaseNode
	Assignee []Assignable
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
	lexer.Type, lexer.Boolean, lexer.Nil, lexer.And, lexer.Or,
	lexer.In, lexer.Break, lexer.Go, lexer.While,
}

// Keywords that can be used as identifiers if they are not followed by specific tokens.
var Modifiers = []lexer.TokenType{
	lexer.Opaque, lexer.Public,
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

type TupleType struct { // TODO: update
	BaseNode
	Values []*TypePair
}

type DestructureTuple struct {
	BaseNode
	Values []*DestructureTypePair
}

type DestructureTypePair struct {
	BaseNode
	Keys []Destructure
	Type Type
}

type ParenType struct {
	BaseNode
	Type Type
}

type FunctionType struct {
	BaseNode
	Parameters *TupleType
	ReturnType Type
}

type GenericType struct {
	BaseNode
	Name       Type
	Parameters []Type
}

type TypePair struct {
	Keys  []Identifier
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
	Label, Identifier Identifier
	Type              Type
	BaseNode
}

type TypeAnnotation struct {
	BaseNode
	Variable *DestructureVars
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
	Module, Alias      Identifier // Alias is nil if no unqualified imports
	Wildcard           bool
	UnqualifiedImports []*UnqualifiedImport
}

type UnqualifiedImport struct {
	BaseNode
	TypeImport        bool
	Wildcard          bool
	Identifier, Alias Identifier
}

type TypeDeclaration interface {
	Statement
	Name() string
	typeDecl()
}

type InterfaceDeclaration struct {
	Identifier     Identifier
	InheritedTypes []Type
	Tag            bool // If no fields
	Fields         []*TypePair
	BaseNode
}

type StructDeclaration struct {
	Identifier     Identifier
	InheritedTypes []Type // Type alias or primitive
	Fields         []*StructField
	BaseNode
}

type StructField struct {
	Names    []Identifier
	Type     Type
	Constant bool
	Value    Expression
	BaseNode
}

type EnumDeclaration struct {
	Identifier Identifier
	Inherited  []Type
	Values     []*EnumItem
	BaseNode
}

type EnumItem struct {
	Identifier Identifier
	Value      Expression
	Parameters []Type
	BaseNode
}

type TypeAliasDeclaration struct {
	Identifier Identifier
	Type       Type
	BaseNode
}

type MapLiteral struct {
	Entries []*MapItem
	BaseNode
}

type MapItem struct {
	Keys            []Expression // if not rest
	Value           Expression
	Rest, Shorthand bool
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
	Identifier    Identifier
	Struct        Identifier
	GenericParams []Identifier
	Parameters    []*FunctionParam
	ReturnType    Type
	Body          []Statement
	Expression    Expression
	BaseNode
}

type FuncAliasDeclaration struct {
	BaseNode

	Struct     Identifier
	Identifier Identifier
	Alias      Expression
}

type FunctionParamName struct {
	BaseNode
	Label, Identifier Identifier
}

type FunctionParam struct {
	Names   []*FunctionParamName
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
	Name Identifier
}

type CallParam struct {
	Label Identifier
	Value Expression
	BaseNode
}

type CallExpression struct {
	Callee Node
	Args   []*CallParam
	BaseNode
}

// StructDotInit is a shorthand constructor for known types
//
//	people: [Person] := [.("John", age: 32), .("Jane", age: 31)]
type StructDotInit struct {
	BaseNode
	Params []*CallParam
}

// An UpdateStatement is a decrement or increment statement. These statements end in
// ++ or --. Unlike other languages such as C, Klar's increment/decrement operators
// are statements rather than expressions.
type UpdateStatement struct {
	Left     Expression
	Operator Operator
	BaseNode
}

// A ForStatement is a loop that executes Body for each item in a list.
//
//	for k, v in <expr>
//	for item in <expr>
//	for 5 { ...repeat 5 times }
type ForStatement struct {
	BaseNode
	Variables  []Destructure
	Expression Expression // When used as while loop or repeat
	Body       []Statement
}

// A WhileStatement executes Body while Condition is true
//
//	while { ...infinite loop }
//	while <expr> - while loop
type WhileStatement struct {
	BaseNode
	Infinite  bool // No condition
	Condition Expression
	Body      []Statement
}

type WhenExpression struct {
	BaseNode
	Subjects []Expression
	Cases    []*WhenCase
}

type WhenCase struct {
	BaseNode
	Options  [][]Expression
	Guard    Expression  // <case> when <expr>
	Body     []Statement // -> <expr> | -> {...}
	BodyExpr Expression
	InBraces bool
}

type LambdaExpression struct {
	BaseNode
	Params   []*TypePair
	Body     []Statement
	ExprBody Expression
}

type Attribute struct {
	BaseNode
	Decorator Identifier
	Args      []*CallParam
}

type RestExpression struct {
	BaseNode
	Left bool
	Expr Expression
}

type RangeExpression struct {
	From, To, Step Expression
	Operator       Operator // First ... or ..<
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

// A ForExpression is a [ForStatement] used as an expression.
// It may reduce when the +=, -=, or = operator is used,
// filter when a block is used, or map when -> is used.
//
//	sum := for i in items += i
//	for [variables] in [iterator] [-> | = | += | -=] [value]
//	for [variables] in [iterator] { block... }
type ForExpression struct {
	BaseNode
	Variables []Destructure
	Iterator  Expression
	Operator  Operator
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
	Node
	Assignable
	destruct()
}

type ListDestructure struct {
	BaseNode
	Tuple  bool
	Values []Destructure
}

// Object or map destructure
type ObjectDestructure struct {
	BaseNode
	Values []*ObjectDestructureEntry
}

// Entry for [ObjectDestructure]
//
//	#{ in.(name, age) }
//	#{ cond: when }
//	#{ data.[{ key }] }
//	#{ data.[{ myKey: key }] }
//	#{ data.[first] }
type ObjectDestructureEntry struct {
	BaseNode
	Alias   Identifier  // before the :
	Object  *Symbol     // after the : or before the .
	Index   Destructure // after the dot
	Default Expression
}

type DestructureVars struct {
	BaseNode
	Values []Assignable
}
