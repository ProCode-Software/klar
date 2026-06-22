package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

func (c *Checker) parseType(expr ast.Type, ctx *Context, flags ...Flag) Type {
	f := parseFlags(flags)
	_ = f
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
		// If the target type hasn't been completed yet, typecheck it
		if Underlying(target.typ) == nil {
			c.checkDeclaration(target)
		}
		if !target.IsTypeName() {
			err := klarerrs.Node(klarerrs.ErrNotAType, expr).
				SetParam("kind", kindOf(target.typ))
			err.Label = "Expected " + quote(name) + " to be a type"
			err.Name = name
			err.AddDetail(quote(name)+" was declared here", target.FilePath(), target.rang)
			c.fileError(err, ctx.File)
			return InvalidType
		}
		return target.typ
	case *ast.MapType:
		return &Map{c.parseType(expr.Key, ctx), c.parseType(expr.Value, ctx)}
	case *ast.FunctionType:
	case *ast.OptionalType:
		return &Optional{c.parseType(expr.Value, ctx)}
	case *ast.GenericType:
	case *ast.ListType:
		return &List{c.parseType(expr.Value, ctx)}
	case *ast.ParenType:
		return c.parseType(expr.Type, ctx)
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
			return ResultNothing
		case ast.PrimitiveError:
			return ErrorType
		default:
			panic(fmt.Sprintf("unhandled primitive type id: %s", expr.Primitive))
		}
	case *ast.RestType:
		// Invalid outside of function. RestType is already explicitly handled
		// when function signatures are checked.
		c.fileError(klarerrs.Node(klarerrs.ErrInvalidRestType, expr), ctx.File)
	case *ast.TupleType:
	case *ast.UnionType:
	case *ast.MethodType:
	default:
		panic(fmt.Sprintf("unhandled type expression: %T", expr))
	}
	return InvalidType
}
