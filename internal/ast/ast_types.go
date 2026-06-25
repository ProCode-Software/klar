// Package ast provides token types for the Klar abstract syntax tree (AST).
package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type BaseNode struct{ Range ranges.Range }

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

type Comment struct {
	BaseNode
	Type  CommentType
	Value string
}

// lexer.LineComment or lexer.BlockComment
type CommentType = lexer.TokenType

// Basic Literals
// =======

type Symbol struct {
	BaseNode
	Identifier string
}

type Discard struct{ BaseNode } // _

type StringLiteral struct {
	BaseNode
	QuoteStyle rune // ' " or `
	// Full string contents including escape literals
	Content string
	// Parts of string split by newlines (at end of segment) and escapes (skipped)
	Fragments []StringFragment
}

type (
	// A StringFragment is a part of a [StringLiteral]. Fragments are split
	// by newlines (at ends of fragments) and escapes.
	StringFragment interface{ StringFrag() }

	// String fragments. Implement [StringFragment].
	EscapeFragment        struct{ Value StringEscape }
	TextFragment          = lexer.TextFragment
	InterpolationFragment struct{ Expression Expression }

	// A StringEscape is an escape sequence inside a [StringLiteral].
	StringEscape interface{ stringEsc() }

	// String escapes. Implement [StringEscape].
	BadEscape         struct{ Source string }
	CharacterEscape   struct{ Character rune }
	UnicodeEscape     struct{ Hex int32 }
	HexadecimalEscape struct{ Hex int32 }
)

type IntegerLiteral struct {
	BaseNode
	Format    lexer.IntFormat
	Value     int64
	Source    string
	Separator bool
}

type BooleanLiteral struct {
	BaseNode
	Value bool
}

type NilLiteral struct{ BaseNode }

type FloatLiteral struct {
	BaseNode
	Value     float64
	Source    string
	Exponent  bool
	Separator bool
}

type RegexLiteral struct {
	BaseNode
	Source    string
	Flags     []byte
	Fragments []StringFragment
	Multiline bool
}

type VersionLiteral struct {
	BaseNode
	Version string
}

// Reference to enum item of a known type
//
//	x: Color := .red
type EnumLiteral struct {
	BaseNode
	Name Identifier
}

// Operations
// =======

type BinaryExpression struct {
	Left, Right Expression
	Operator    Operator
	BaseNode
}

type UnaryExpression struct {
	Operator Operator // [lexer.Minus] or [lexer.Not]
	Right    Expression
	BaseNode
}

// A RelationalExpression is a [BinaryExpression] with a relational operator
// or a comparison chain.
type RelationalExpression struct {
	BaseNode
	Expressions []Expression
	Operators   []Operator // Should be len(Expressions) - 1
}

// [expr]!
type AssertExpression struct {
	BaseNode
	Expression Expression
}

// Accessors & Collections
// =======

type IndexExpression struct {
	Object, Property Expression // Property is [*Symbol] if !Computed
	Computed         bool       // If square bracket '['
	BaseNode
}

// A list slice expression
//
//	array[low..<high]
//	array[low...high]
//	array[low...]
//	array[..<high]
//	array[...high]
//	array[...] // copy
type SliceExpression struct {
	Object   Expression
	From, To Expression
	Operator Operator
	BaseNode
}

type RangeExpression struct {
	From, To, Step Expression
	Operator       Operator // First ... or ..<
	BaseNode
}

type RestExpression struct {
	BaseNode
	Expression Expression // Can be nil in when cases
}

// More complex literals
// =======

type CallParam struct {
	Label     *Identifier
	Value     Expression
	Shorthand bool // Whether shorthand was used. Label != nil
	BaseNode
}

// A function call
type CallExpression struct {
	Callee Expression
	Args   []*CallParam
	BaseNode
}

type ParenExpression struct {
	Expression Expression
	BaseNode
}

type ListLiteral struct {
	BaseNode
	Items []Expression
}

type MapLiteral struct {
	Entries []*MapItem
	BaseNode
}

// If Rest is true, len(Keys) should be 0.
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

// StructDotInit is a shorthand constructor for known types
//
//	people: [Person] := [.("John", age: 32), .("Jane", age: 31)]
type StructDotInit struct {
	BaseNode
	Params []*CallParam
}

type ListCastExpression struct {
	BaseNode
	Type Type // Item type
	Args []*CallParam
}

type MapCastExpression struct {
	BaseNode
	KeyType, ValueType Type
	Args               []*CallParam
}

// Control Expressions
// =======

type WhenExpression struct {
	BaseNode
	Subjects []Expression
	Label    *Identifier
	Cases    []*WhenCase
}

type WhenCase struct {
	BaseNode
	Options [][]Expression // cases -> subjects
	Guard   Expression     // <case> if <expr>
	Braces  bool
	Body    Node // [*Block], [Statement], or [Expression]. Syntax: -> <expr> | -> {...}
}

// In string interpolations for pattern matching in when blocks.
type StringTypeMatch struct {
	BaseNode
	Name Identifier
	Type Type
}

type LambdaExpression struct {
	BaseNode
	Params  []*AssignableTypePair
	InParen bool
	Block   *Block
	Expr    Expression
}

type PipelineExpression struct {
	Steps []Node // [Expression] or [*ReturnStatement]
	BaseNode
}

// A pipeline step may be an assignment or method call (ignoring return values)
type ObjectPipeline struct {
	BaseNode
	Object Expression
	Steps  []Node // Assignment or method call
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
	Variables []*AssignableTypePair
	Iterator  Expression
	Operator  Operator
	Value     Expression
	Block     *Block
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
//	go { ...body }
type GoExpression struct {
	BaseNode
	Expression Expression // Is a *CallExpression if valid
	Body       *Block     // If block
}

// where 'fn' returns a Result, returns from the enclosing function if
// it returns an error value. Only available in function bodies.
// try fn() -- must be function call
type TryExpression struct {
	BaseNode
	Expression Expression // Is a *CallExpression if valid
}

// Statements
// =======

type Attribute struct {
	BaseNode
	Name Identifier
	Args []*CallParam
}

// An ExpressionStatement is an expression used as a statement.
type ExpressionStatement struct {
	BaseNode
	Expression Expression
}

type Block struct {
	BaseNode
	Body []Statement
}

type ReturnStatement struct {
	Value Expression // Can be nil
	BaseNode
}

// Continues a loop. In other languages, this is usually 'continue'.
// Used for [ForStatement], [WhileStatement], and [WhenExpression] statements.
type NextStatement struct {
	BaseNode
	Label *Identifier
}

// Breaks a [ForStatement], [WhileStatement], or [WhenExpression].
// In other languages, this is usually 'break'.
type StopStatement struct {
	BaseNode
	Label *Identifier
}

// A ForStatement is a loop that executes Body for each item in a list.
//
//	for k, v in <expr> [:label]
//	for item in <expr>
//	for 5 { ...repeat 5 times }
type ForStatement struct {
	BaseNode
	Variables  []*AssignableTypePair // Pairs don't have default values
	Label      *Identifier
	Expression Expression // When used as while loop or repeat
	Body       *Block
}

// A WhileStatement executes Body while Condition is true
//
//	while { ...infinite loop }
//	while <expr> - while loop
type WhileStatement struct {
	BaseNode
	Condition Expression
	Label     *Identifier
	Body      *Block
}

// Examples:
//
//		import klar.http
//		import klar.http.*
//		import klar.regex.{RegEx}
//		import klar.http.requests.{get as fetch}
//	 import globalThis = klar.js
type ImportStatement struct {
	BaseNode
	Alias              *Identifier // Can be nil
	Module             []string
	Wildcard           bool
	UnqualifiedImports []*IdentifierPair
}

type PublicDeclaration struct {
	BaseNode
	Declaration Statement
}

type VariableDeclaration struct {
	BaseNode
	Variables    []Assignable
	Values       []Expression // Either 1 item or len(Variables)
	ExplicitType Type
}

// Reports whether the variable declaration has a single RHS expression
// with multiple variables on the LHS.
func (vd *VariableDeclaration) IsSingleRHS() bool {
	return len(vd.Values) == 1 && len(vd.Variables) > 1
}

type AssignmentStatement struct {
	BaseNode
	Assignee []Assignable
	Operator Operator
	Values   []Expression // Either 1 item or len(Variables)
}

// A FunctionDeclaration is a basic Klar function or method declaration.
//
//	func run()
//	func Parser.run()
//	func (p: Parser).run()
type FunctionDeclaration struct {
	Identifier    Identifier
	SelfType      *Identifier // Struct.Identifier
	SelfName      *Identifier // If (p: Parser) is used instead of Parser
	GenericParams []Identifier
	Parameters    []*FunctionParam
	ReturnType    Type
	Body          *Block
	Expression    Expression
	BaseNode
}

type FunctionParam struct {
	Names   []*IdentifierPair
	Type    Type
	Default Expression
	BaseNode
}

// func x = y
// func Parser.x = .y
// func x = module.y
type FuncAliasDeclaration struct {
	BaseNode
	Struct     *Identifier
	Identifier Identifier
	Target     Expression
}

// Type Declarations
// =======

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
	Items          []*InterfaceItem
	BaseNode
}

type TagDeclaration struct {
	Identifier     Identifier
	InheritedTypes []Type
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
	Value      Expression
	Attributes []*Attribute
	BaseNode
}

type EnumDeclaration struct {
	Identifier Identifier
	Generics   []Identifier
	Inherited  []Type
	Values     []*EnumItem
	BaseNode
}

type EnumItem struct {
	Identifier Identifier
	Value      Expression
	Parameters *TupleType
	Attributes []*Attribute
	BaseNode
}

type TypeAliasDeclaration struct {
	Identifier Identifier
	Type       Type
	BaseNode
}

// Types
// =====

// A PrimitiveType is the name of a Klar-builtin type
type PrimitiveType struct {
	BaseNode
	Primitive PrimitiveTypeName
}

// A TypeAlias is a non-primitive type name
type TypeAlias struct {
	BaseNode
	Identifier string
}

// A QualifiedTypeAlias is similar to [TypeAlias] but is qualified by a namespace.
type QualifiedTypeAlias struct {
	BaseNode
	Namespace, Identifier Identifier
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

type MapType struct {
	BaseNode
	Key, Value Type
}

// Function parameters only
type RestType struct {
	BaseNode
	Value Type
}

// A TupleType is a tuple of types.
type TupleType struct {
	BaseNode
	Values []*TypePair
}

// A ParenType is a type enclosed in parentheses.
type ParenType struct {
	BaseNode
	Label Identifier // Optional label
	Type  Type
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

type InterfaceItem struct {
	*TypePair
	Attributes []*Attribute
}

type UnionType struct {
	BaseNode
	Options []Type
}

// Can only be present in [*InterfaceItem]
type MethodType struct {
	BaseNode
	ReturnType    Type
	GenericParams []Identifier
	Parameters    []*MethodParam
}

type MethodParam struct {
	Names []*IdentifierPair
	Type  Type
	BaseNode
}

// Assignments & Destructuring
// =======

type Assignable interface {
	Node
	Expression
	assignable()
	// If an item implements [Destructurable], a [*Symbol] is yielded,
	// otherwise the [Assignable] expression is returned. Items that aren't
	// assignable yield a nil [Assignable] and a non-nil [BadExpression].
	Every(func(Assignable, *BadExpression) bool) bool
}

type Destructurable interface {
	Assignable
	destruct()
}

// Values that can be used in VariableDeclaration implement Destructure.
//
// Deprecated: Not used
//
//	(a, b) | [a, b] | #{ a, b } | a
type Destructure interface {
	Node
	Expression
	Assignable
	destruct()
}

// Destructures a list or tuple
//
// Deprecated: Not used
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
// Deprecated: Not used
//
//	#{ name, age } := person
//	#{ kind: type, info.{color} } := animal
type ObjectDestructure struct {
	BaseNode
	Values []*ObjectDestructureEntry
}

// Entry for [ObjectDestructure]
//
// Deprecated: Not used
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

// Pairs & Helpers
// ========

type TypePair struct {
	Keys  []Identifier
	Value Type
	BaseNode
}

type ExpressionPair struct {
	BaseNode
	Key, Value Expression
}

type IdentifierPair struct {
	BaseNode
	Label, Name Identifier
}

type AssignableTypePair struct {
	BaseNode
	Keys  []Assignable
	Type  Type
	Value Expression
}
