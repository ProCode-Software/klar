package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) ParseFunction(d ast.FunctionDeclaration, ctx context) (f types.Function) {
	f.Params = make([]types.Param, len(d.Parameters))
	f.Return = c.ParseType(d.ReturnType, ctx)
	return f
}

func (c *Checker) parseFuncDecls(funcs []ast.FunctionDeclaration, ctx context) {
	for _, f := range funcs {
	}
}
