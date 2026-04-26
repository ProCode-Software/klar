package analysis

import (
	"fmt"
	"iter"
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
)

func (c *Checker) checkVarDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
}

func (c *Checker) checkConstDecl(o *Object, decl *DeclarationInfo) {
	_ = decl.node.(*ast.VariableDeclaration)
}

func DestructureNames(a ast.Assignable) iter.Seq[ast.Identifier] {
	switch a := a.(type) {
	case *ast.Symbol:
	case *ast.Discard:
	case *ast.ListLiteral:
	case *ast.TupleLiteral:
	case *ast.MapLiteral:
	case *ast.BadExpression:

	default:
		panic(fmt.Sprintf("unhandled destructure node: %T", a))
	}
	return func(yield func(ast.Identifier) bool) {}
}

// IsConst returns true if the given name is a constant name
// (all uppercase). Digits and underscores are allowed.
func IsConst(name string) bool {
	for _, r := range name {
		// Some characters, like CJK, are neither upper nor lower case. Allow them.
		if unicode.IsLower(r) && !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}
