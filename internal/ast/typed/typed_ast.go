package typed

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type (
	Node interface{ node() }
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

type FunctionDecl struct {
	BaseNode
	Name       string
	Params     []FuncParam
	ReturnType Type
}

type VariableDecl struct {
	BaseNode
	Constant bool
	Name     string
	Type     Type
	Value    Expression
}

type EnumDecl struct {
	BaseNode
	ValueType Type
	Items     map[string]any
}

type TypeAliasDecl struct {
	BaseNode
	Name string
	Type Type
}

type StructInterfaceDecl struct {
	BaseNode
	Inherited
	Interface bool
	Name      string
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

type Statement struct {
	BaseNode
}

type FuncParam struct {
	BaseNode
	Label, Var string
	Type       Type
}
