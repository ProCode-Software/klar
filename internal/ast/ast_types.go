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
	QuoteStyle rune // ' " or `
	// Number of QuoteStyle after '@' if it was used, otherwise 0
	QuoteCount int
	// Full string contents including escape literals
	Content string
	Escapes map[lexer.Position]StringEscape
	// Parts of string split by newlines (at end of segment) and escapes (skipped)
	Segments []string
}

type IntegerLiteral struct {
	BaseNode
	Format    lexer.IntegerFormat
	Value     int64
	Source    string
	Separator bool
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
	Value     float64
	Source    string
	Exponent  bool
	Separator bool
}

type RegexLiteral struct {
	BaseNode
	Source     string
	Flags      []rune
	Multiline  bool
	QuoteCount int // Number of '/' if '@' was used. Zero if it wasn't
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

// Multi comparisions using ==, !=, >=, <=, >, or <
type RelationalExpression struct {
	BaseNode
	Expressions []Expression
	Operators   []Operator // Should be len(Expressions) - 1
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
	lexer.And, lexer.Await, lexer.Boolean, lexer.Break, lexer.For, lexer.Func,
	lexer.Go, lexer.Import, lexer.In, lexer.Next, lexer.Nil, lexer.Or,
	lexer.Return, lexer.Type, lexer.When, lexer.While,
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

type TupleType struct {
	BaseNode
	Values []*TypePair
	Single bool // If 1 item without trailing comma
}

// Used for lambda parameters
type DestructureTuple struct {
	BaseNode
	Values []*DestructureTypePair
}

type DestructureTypePair struct {
	BaseNode
	Keys  []Destructure
	Type  Type
	Value Expression
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

type InterfaceItem struct {
	TypePair
	Attributes []*Attribute
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
	Names [][2]Identifier // label, name
	Type  Type
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
//	import klar.regex.{RegEx}
//	import fetch = klar.http.requests.{get}
type ImportStatement struct {
	BaseNode
	Module, Alias      Identifier // Alias is nil if no unqualified imports
	Wildcard           bool
	UnqualifiedImports []*UnqualifiedImport
}

type UnqualifiedImport struct {
	BaseNode
	Wildcard          bool
	Identifier, Alias Identifier
}

type TypeDeclaration interface {
	Statement
	Name() string
	typeDecl()
}

type ModifierDeclaration interface {
	Statement
	modif()
}

type InterfaceDeclaration struct {
	Identifier     Identifier
	InheritedTypes []Type
	Tag            bool // If no fields
	Fields         []*InterfaceItem
	BaseNode
}

type StructDeclaration struct {
	Identifier     Identifier
	InheritedTypes []Type // Type alias or primitive
	Fields         []*StructField
	BaseNode
}

type StructField struct {
	Names      []Identifier
	Type       Type
	Constant   bool
	Value      Expression
	Attributes []*Attribute
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
	Parameters []*TypePair
	Attributes []*Attribute
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

// Continues a loop. In other languages, this is usually 'continue'.
// Used for [ForStatement], [WhileStatement], and [WhenExpression] statements.
type NextStatement struct{ BaseNode }

// Breaks a [ForStatement], [WhileStatement], or [WhenExpression].
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

// A list slice expression
//
//	array[low:high]
//	array[low:]
//	array[:high]
//	array[:] -- copy
type SliceExpression struct {
	Object      Node
	Start, High Expression
	BaseNode
}

// Reference to enum item of a known type
//
//	x: Color := .red
type EnumLiteral struct {
	BaseNode
	Name Identifier
}

type CallParam struct {
	Label Identifier
	Value Expression
	BaseNode
}

// A function call
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
	Variables  []*DestructureTypePair
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
	Options  [][]Expression // cases -> subjects
	Guard    Expression     // <case> when <expr>
	Body     []Statement    // -> <expr> | -> {...}
	BodyExpr Node
	InBraces bool
}

type WhenCanCase struct {
	BaseNode
	Operator Operator
	Type     Type
	Params   []*CallParam
}

type LambdaExpression struct {
	BaseNode
	Params   []*DestructureTypePair
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
	Left       bool
	Expression Expression
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
	Expression Expression
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
	Variables []*DestructureTypePair
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

// Destructures a list or tuple
//
//	(a, b) := x
//	[a, b] := x
type ListDestructure struct {
	BaseNode
	Tuple  bool
	Values []Destructure
}

// Destructures a struct or map
//
//	#{ name, age } := person
//	#{ kind: type, info.{color} } := animal
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

// For parsing variable declarations and assignments
type DestructureVars struct {
	BaseNode
	Values []Assignable
}

// Declares a struct that can only be initialized within a module
// or an interface that can only be implemented within a module
type OpaqueDeclaration struct {
	BaseNode
	Declaration TypeDeclaration
}

// Waits for one or more tasks to complete
//
//	await task
//	await [t1, t2]
//	await (t1, t2)
type AwaitExpression struct {
	BaseNode
	Expression Expression
}

// Spawns an asynchronous task
//
//	go fn()
//	go (a, b, c)
//	go [a, b, c]
//	go { ...body }
type GoExpression struct {
	BaseNode
	Expression Expression
	Body       []Statement // If block
}
