package analysis

import "github.com/ProCode-Software/klar/internal/ast"

func (c *Checker) checkVarDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
}

func (c *Checker) checkConstDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
}
