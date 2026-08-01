package printer

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
)

func (p *printer) printType(typ ast.Type) {
	switch typ := typ.(type) {
	case *ast.TypeAlias:
	case *ast.QualifiedTypeAlias:

	case *ast.MapType:
	case *ast.ListType:
	case *ast.GenericType:
	case *ast.UnionType:
	case *ast.OptionalType:
	case *ast.TupleType:
	case *ast.FunctionType:
	default:
		panic(fmt.Sprintf("unhandled type: %T", typ))
	}
}
