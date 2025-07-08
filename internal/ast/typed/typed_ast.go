package typed

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type (
	Node interface {
		At() ranges.Range
	}
	Type = types.Type
)

type BaseNode struct {
	Position ranges.Range
}

type Program struct {
	BaseNode
	Imports []ast.ImportStatement
	Exports []Declaration
	Context
}

type Context struct {
	Types      []TypeDecl
	Functions  []FunctionDecl
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
type BaseDecl struct {
	Name string
}

type FunctionDecl struct {
	BaseNode
	Name       string
	Params     []FuncParam
	ReturnType Type
	Body       Context
}
type FuncParam struct {
	BaseNode
	Label, Var string
	Type       Type
	Default    Expression
}

type VariableDecl struct {
	BaseNode
	Idents          Expression
	ConstantIndices []int
	Type            Type
	Value           Expression
}

type EnumDecl struct {
	BaseNode
	BaseDecl
	ValueType Type
	Items     map[string]EnumItem
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
	Methods   map[string]types.Overloads
}

type Inherited struct {
	Inherits         []string
	InheritedFields  map[string]Type
	InheritedMethods map[string]types.Overloads
	Implements       []Type
}

type Expression struct {
	BaseNode
	Type Type
	Expr ast.Expression
}

type Statement = ast.Statement
