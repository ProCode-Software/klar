package codegen

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertStatement(stmt ast.Statement, ctx *analysis.Context) jsir.Statement {
	switch stmt := stmt.(type) {
	case *ast.VariableDeclaration:
		_ = stmt
	}
	return &jsir.FunctionDeclaration{Name: randomName(6), Body: &jsir.Block{}}
}
