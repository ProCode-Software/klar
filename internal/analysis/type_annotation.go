package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
)

// TODO: shoul we accept a file id?
func (c *Checker) parseType(expr ast.Type, ctx *Context) Type {
	switch expr := expr.(type) {
	case *ast.BadExpression:
	case *ast.TypeAlias:
	case *ast.MapType:
	case *ast.FunctionType:
	case *ast.OptionalType:
	case *ast.GenericType:
	case *ast.ListType:
	case *ast.ParenType:
	case *ast.QualifiedTypeAlias:
	case *ast.PrimitiveType:
	case *ast.RestType:
	// Invalid outside of function. RestType is already explicitly handled
	// when function signatures are checked.
	case *ast.TupleType:
	case *ast.UnionType:
	case *ast.MethodType:
	default:
		panic(fmt.Sprintf("unhandled type expression: %T", expr))
	}
	return InvalidType
}
