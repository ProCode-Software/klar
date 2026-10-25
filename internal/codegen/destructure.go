package codegen

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertDestructure(dest ast.Destructurable, nameMangle bool) jsir.Destructure {
	switch dest := dest.(type) {
	case *ast.Discard:
		return nil
	case *ast.Symbol:
		return escapedSymbol(dest.Identifier, nameMangle)
	case *ast.ListLiteral:
	case *ast.TupleLiteral:
	default:
		panic(fmt.Sprintf("unhandled ast.Destructurable node: %T", dest))
	}
	return nil
}
