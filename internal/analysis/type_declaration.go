package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
)

// o's Type should be [*TypeName]
func (c *Checker) checkCustomTypeDecl(o *Object, decl *DeclarationInfo) {
	node := decl.node.(ast.TypeDeclaration)
	switch node := node.(type) {
	case *ast.StructDeclaration:
		c.checkStructDecl(o, node, decl.file)
	case *ast.EnumDeclaration:
		c.checkEnumDecl(o, node, decl.file)
	case *ast.TypeAliasDeclaration:
		c.checkTypeAlias(o, node, decl.file)
	case *ast.InterfaceDeclaration:
		if node.Tag {
			o.typ.(*TypeName).Type = TagType
			return
		}
		c.checkInterfaceDecl(o, node, decl.file)
	default:
		panic(fmt.Sprintf("unknown type declaration: %T", node))
	}
}
