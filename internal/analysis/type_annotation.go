package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

func (c *Checker) parseType(expr ast.Type, ctx *Context) Type {
	switch expr := expr.(type) {
	case nil:
		panic("parseType(nil)")
		// return InvalidType
	case *ast.BadExpression:
		return InvalidType
	case *ast.TypeAlias:
		name := expr.Identifier
		target := ctx.LookupRecursive(name)
		if target == nil {
			c.fileError(klarerrs.Undefined(name, expr.GetRange()), ctx.File)
			return InvalidType
		}
		return target.typ
	case *ast.MapType:
	case *ast.FunctionType:
	case *ast.OptionalType:
	case *ast.GenericType:
	case *ast.ListType:
	case *ast.ParenType:
	case *ast.QualifiedTypeAlias:
	case *ast.PrimitiveType:
		switch expr.Primitive {
		case ast.PrimitiveInt:
			return IntType
		case ast.PrimitiveString:
			return StringType
		case ast.PrimitiveBool:
			return BoolType
		case ast.PrimitiveAny:
			return AnyType
		case ast.PrimitiveFloat:
			return FloatType
		case ast.PrimitiveNothing:
			return NothingType
		case ast.PrimitiveResult:
			return nil
		case ast.PrimitiveError:
			return ErrorType
		}
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
