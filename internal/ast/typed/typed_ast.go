package typed

import (
	"github.com/ProCode-Software/klar/internal/ast"
)

type (
	Type     ast.Type
	BaseNode ast.BaseNode
)

type Node interface{
	BaseNode
}

type Program struct {
	BaseNode
	Body []Statement
}

type Expression struct {
	BaseNode
	Type Type
	Expr ast.Expression
}

type Statement struct{
	BaseNode
}

type VarDecl struct {
	BaseNode
	Constant bool
	Name     string
	Type     Type
	Value    Expression
}

type FuncParam struct {
	BaseNode
	Label, Var string
	Type       Type
}

type FuncDecl struct {
	BaseNode
	Struct     Type
	Name       string
	Params     []FuncParam
	ReturnType Type
}
