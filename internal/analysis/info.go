package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type Info struct {
	ExprTypes map[ast.Expression]TypeAndValue
	
}

type TypeAndValue struct {
	Type       Type
	Kind       ExprKind
	ConstValue ConstValue
}
