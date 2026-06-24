package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type Info struct {
	Expressions map[ast.Expression]*Expr
}
