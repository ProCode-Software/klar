package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
)

func (c *Checker) checkTypeDecl(o *Object, decl *DeclarationInfo) {
	node := decl.node.(ast.TypeDeclaration)
	switch node := node.(type) {
	case *ast.StructDeclaration:
	case *ast.EnumDeclaration:
	case *ast.TypeAliasDeclaration:

	case *ast.InterfaceDeclaration:
		if node.Tag {
			typ := o.typ.(*TypeName)
			typ.Underlying = TagType
			return
		}

	default:
		panic(fmt.Sprintf("unknown type declaration: %T", node))
	}
}
