package analysis

import (
	"fmt"
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
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
		// If the type is generic, ensure it has type parameters
		if min, max := numGenerics(target.typ); min > 0 && f&genericLHS == 0 {
			err := genericParamsCountError(
				klarerrs.ErrGenericParamsRequired, name, expr.Range, min, max,
			)
			c.fileError(err, ctx.File)
			return InvalidType
		}
		return target.typ
	case *ast.MapType:
		return &Map{c.parseType(expr.Key, ctx), c.parseType(expr.Value, ctx)}
	case *ast.FunctionType:
	case *ast.OptionalType:
		inner := c.parseType(expr.Value, ctx)
		// Don't chain optionals. TODO: Check that the user doesn't do this
		if _, ok := inner.(*Optional); ok {
			return inner
		}
		return &Optional{inner}
	case *ast.GenericType:
		return c.parseGenericType(expr, ctx)
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
			// TODO: Task
		default:
			panic(fmt.Sprintf("unhandled primitive type id: %s", expr.Primitive))
		}
	case *ast.RestType:
		// Invalid outside of function. RestType is already explicitly handled
		// when function signatures are checked.
		c.fileError(klarerrs.Node(klarerrs.ErrInvalidRestType, expr), ctx.File)
	case *ast.TupleType:
		tup := Tuple{}
		for _, pair := range expr.Values {
			item := c.parseType(pair.Value, ctx)
			for range max(len(pair.Keys), 1) {
				tup = append(tup, item)
			}
		}
		return tup
	case *ast.UnionType:
		return c.checkUnionType(expr, ctx)
	case *ast.MethodType:
	default:
		panic(fmt.Sprintf("unhandled type expression: %T", expr))
	}
	return InvalidType
}

func numGenerics(t Type) (min, max int) {
	switch t.Kind() {
	case KindResult:
		return 0, 2
	case KindEnum:
		n := len(Underlying(t).(*Enum).Generics)
		return n, n
	case KindTask:
		return 1, 1
	default:
		return 0, 0
	}
}

func genericParamsCountError(
	code klarerrs.Code, name string, r ranges.Range,
	min, max int,
) *klarerrs.Error {
	err := klarerrs.Range(code, r)
	var countRange string
	if min == max {
		countRange = strconv.Itoa(min)
	} else {
		countRange = fmt.Sprintf("%d-%d", min, max)
	}
	err.Name = name
	err.SetParam("count", countRange)
	err.Label = "Type " + quote(name) + " requires " + countRange + " generic parameter"
	if max != 1 {
		err.Label += "s"
	}
	return err
}

func (c *Checker) parseGenericType(expr *ast.GenericType, ctx *Context) Type {
	lhs := c.parseType(expr.Name, ctx, genericLHS)
	// Validate the count of required generic parameters
	minCt, maxCt := numGenerics(lhs)
	if len(expr.Parameters) < minCt || len(expr.Parameters) > maxCt {
		err := genericParamsCountError(
			klarerrs.ErrInvalidGenericCount, lhs.String(), expr.Range, minCt, maxCt,
		)
		if maxCt == 0 {
			// Type doesn't take any generic params
			err.Code = klarerrs.ErrNonGenericType
			err.Label = "Can't pass generic parameters to " + quote(lhs.String())
		}
		c.fileError(err, ctx.File)
		return lhs
	}
	// Parse the passed generic types
	params := make([]Type, len(expr.Parameters))
	for i, param := range expr.Parameters {
		params[i] = c.parseType(param, ctx)
	}
	// Apply the generics and return
	switch lhs.Kind() {
	case KindResult:
		return &Result{params[0], params[1]}
	case KindEnum:
		return lhs // TODO: Substitute generic params
	case KindTask:
		return &Task{params[0]}
	default:
		panic(fmt.Sprintf("invalid or unhandled generic type LHS: %T", lhs))
	}
}

func (c *Checker) checkUnionType(expr *ast.UnionType, ctx *Context) Type {
	types := make([]Type, len(expr.Options))
	for i, t := range expr.Options {
		types[i] = c.parseType(t, ctx)
	}
	return &Union{Types: types}
}
