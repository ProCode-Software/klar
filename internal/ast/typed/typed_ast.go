package typed

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/attribute"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type (
	Type      = types.Type
	Statement interface {
		Node
		stmt()
	}
	Node interface {
		GetRange() ranges.Range
	}
	BaseNode = ast.BaseNode
	BaseDecl struct {
		Name string
	}
	OverloadList = []*FuncOverload
)

// A Program is a Klar program ready to be compiled or interpreted.
type Program struct {
	BaseNode
	Imports    []*ast.ImportStatement
	Exports    []Declaration
	Attributes map[Declaration]attribute.Attribute
	Context
}

// A Context is a block with its own declarations and scope.
// A Context may contain type and function declarations outside the top level.
type Context struct {
	Types      []TypeDecl
	Functions  []*FunctionDecl
	Statements []Statement
}

// ==========================
// Declarations
// ==========================

type Declaration interface {
	Node
	GetName() string
}
type TypeDecl interface {
	Declaration
	GetType() Type
}

// ==========================
// Type Declarations
// ==========================
type EnumDecl struct {
	BaseNode
	BaseDecl
	ValueType      Type
	Inherited      []string
	InheritedItems map[string]*EnumItem
	Items          map[string]*EnumItem
}

type EnumItem struct {
	BaseNode
	Value  any
	Params []Type
}

type TypeAliasDecl struct {
	BaseNode
	BaseDecl
	Type Type
}

type StructInterfaceDecl struct {
	BaseNode
	BaseDecl
	Inherited
	Interface bool
	Fields    map[string]Type
	Methods   map[string][]OverloadList
}

type Inherited struct {
	Inherits         []string
	InheritedFields  map[string]Type
	InheritedMethods map[string][]OverloadList
	Implements       []Type
}

// ==========================
// Normal Declarations
// ==========================
type FunctionDecl struct {
	BaseNode
	BaseDecl
	Name      string
	Overloads []OverloadList
}
type FuncOverload struct {
	BaseNode
	Params     []*FuncParam
	ReturnType Type
	Body       *Context
}
type FuncParam struct {
	BaseNode
	Label, Var string
	Type       Type
	Variadic   bool
	Default    Expression
}
type VariableDecl struct {
	BaseNode
	Idents          Expression
	ConstantIndices []int
	Type            Type
	Value           Expression
}

// ==========================
// Expressions
// ==========================
type Expression struct {
	BaseNode
	Type Type
	Expr ast.Expression
}
