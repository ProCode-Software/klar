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
	case *ast.TagDeclaration:
	case *ast.EnumDeclaration:
	case *ast.StructDeclaration:
	case *ast.FunctionDeclaration:
	case *ast.FuncAliasDeclaration:
	case *ast.InterfaceDeclaration:
	case *ast.TypeAliasDeclaration:
	}
	return nil
}
