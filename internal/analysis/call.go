package analysis

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (c *Checker) checkCallExpr(expr *ast.CallExpression, t *Expr) {
	lhs := t.NewChild()
	if lhsExpr, ok := expr.Callee.(*ast.Symbol); ok {
		// Allow types for initializers
		c.checkSymbolExpr(lhsExpr, true, lhs)
		if t.Type == nil {
			t.Type = InvalidType
		}
		c.Info.Expressions[expr] = t
	} else {
		c.checkExpr(expr.Callee, lhs)
	}
	if lhs.Kind() == InvalidType {
		t.Type = InvalidType
		return
	}
	canCall := true
	switch fn := UnderlyingTypeName(lhs.Type, true).(type) {
	case *Overload, *Lambda, *TypeName, *EnumFunction:
	case *Function:
		if isTODO(fn) {
			t.Root.mode |= todoExpr
		}
		// TODO: This is temporary and will be removed when generic inference
		// is implemented
		if isCloneBuiltin(fn) && len(expr.Args) >= 1 {
			p1 := c.checkExprFrom(expr.Args[0].Value, t)
			t.Type = p1.Type
			return
		}
	case *UntypedInit:
		if fn.kind == KindEnum && fn.Params == nil {
			enum := expr.Callee.(*ast.EnumLiteral)
			calledInit := &UntypedInit{kind: KindEnum, Node: enum, Params: expr.Args}
			t.Type = calledInit
			c.queue(func() {
				c.checkEnumParams(expr, t)
			}, true)
			return // Won't check params now
		}
		canCall = false
	default:
		canCall = false
		if obj, ok := lhs.Type.(*Object); ok && obj.IsTypeName() {
			if _, ok := obj.TypeName().Type.(*TypeAlias); ok {
				// Allow all type aliases to be called for type casts. There isn't
				// always an expression syntax for every type (such as unions), so
				// if the user aliases them, they can cast.
				// 	type MyUnion = String | Int
				// 	union := MyUnion("hello")
				canCall = true
			}
		}
	}
	if !canCall {
		// Not a function (or initializer)
		err := klarerrs.Node(klarerrs.ErrNotAFunction, expr.Callee)
		// If the user tries to call an enum item that doesn't take parameters,
		// show a different error
		if ei, ok := Underlying(lhs.Type).(*EnumRef); ok {
			err.Code = klarerrs.ErrEnumItemNoParams
			err.Label = fmt.Sprintf("Can't pass parameters to %s.%s", ei.Enum.Name, ei.Name)
			err.Name = ei.Name
		} else {
			typ := quoteAka(lhs.Type)
			err.Label = "This callee has type " + typ + " and can't be called"
			err.Name = typ
		}
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	}
	c.checkCallArgs(lhs.Type, expr.Args, t)
}

func (c *Checker) checkCallArgs(lhs Type, args []*ast.CallParam, t *Expr) (overload Type) {
	var name string
	if obj, ok := lhs.(*Object); ok {
		name = obj.Name
	}
	switch fn := UnderlyingTypeName(lhs, true).(type) {
	case *TypeName, *Map, *List: // List/map cast
		t.Type = lhs
		// For type casts `T(v)`, `v` must be compatible with `T`.
		//
		// Or, if `v` is an implementation of `T` - `V(t)`, `V?` is returned.
		// `t` must be a union, tag, or interface. If `t` is an optional or Result,
		// show a hint that there's a more idiomatic way to check.
		//
		// If we have `T(t)`, show an error that the cast is redundant.
		typ := fn
		if tn, ok := fn.(*TypeName); ok {
			typ = tn.Type
		}

		// Primitive initializer
		var isPrimitive bool
		switch und := typ.(type) {
		case Kind:
			isPrimitive = true
			typ = builtinModule.Context.Lookup(und.String()).TypeName().
				Type.(*bootstrapType).asDeclared
		case *bootstrapType:
			isPrimitive = true
			typ = und.asDeclared
		}
		_ = isPrimitive

		var firstArg Type
		if typ, ok := Underlying(typ).(*Struct); ok {
			// Structs can be initialized in other ways, not just casts
			// TODO: Disallow using default initializers for builtins
			res, isCast := c.checkStructInitializer(typ, name, args)
			if !isCast {
				t.Type = lhs // Preserve type name
				// TODO: res may be an optional. Make it return a kind instead?
				return
			}
			firstArg = res
		}
		c.checkTypeCast(typ, name, args, firstArg, t)
	case *EnumRef:
		t.Type = lhs
	case *Lambda:
		c.checkLambdaParams(fn, args, t)
	case *Function:
		t.Type = fn.Return
		pset := c.inferCallParams(args, t)
		ov, err := c.resolveOverload(fn.Overloads, pset)
		if err != nil {
			err.Range = ranges.FromSlice(args)
			c.fileError(err, t.FileID())
		}
		_ = ov
	case *Overload:
		t.Type = fn.Return
	default:
		panic(fmt.Sprintf(
			"checkCallArgs: unhandled LHS type after being handled by checkCallExpr: %T (underlying: %T)",
			lhs, fn,
		))
	}
	return lhs // TODO
}

func (c *Checker) checkArity(args []*ast.CallParam, arity Arity, got int, fid FileID) bool {
	if arity.InRange(got) {
		return true
	}
	err := klarerrs.Slice(klarerrs.ErrWrongParamCount, args)
	err.SetParam("got", got)
	if got < arity.MinParams {
		err.SetParam("notEnough", true)
	}
	switch got {
	case 0:
		err.Label = "No parameters were provided"
	case 1:
		err.Label = "1 parameter was provided"
	default:
		err.Label = strconv.Itoa(got) + " parameters were provided"
	}
	c.fileError(err, fid)
	return false
}

func (c *Checker) checkLambdaParams(fn *Lambda, args []*ast.CallParam, t *Expr) {
	t.Type = fn.Return
	arity := fn.Arity()
	if !c.checkArity(args, arity, len(args), t.FileID()) {
		return
	}
	for i := 0; i < len(args); i++ {
		var (
			arg        = args[i]
			isVariadic = fn.Variadic && i >= len(fn.Params)
			expType    = fn.Params[min(i, len(fn.Params)-1)]
		)
		if isVariadic {
			expType = expType.(*List).Elem
		}
		// Allowed if tuple or in variadic param
		if rest, ok := arg.Value.(*ast.RestExpression); ok {
			var (
				rhs      = c.checkExprFrom(rest.Expression, t.NewChild())
				rhsRang  = rest.Expression.GetRange()
				itemType Type
			)
			switch rhs.Kind() {
			case KindTuple:
				tup := As[*Tuple](rhs.Type)
				switch {
				case tup.Len() < 2:
					c.fileError(makeTupleSpreadCountError(rest, tup.Len(), false), t.FileID())
				case isVariadic:
					commonType, err := canSpreadTupleIntoList(tup)
					if err != nil {
						err.Range = rhsRang
						c.fileError(err, t.FileID())
						break
					}
					if !Compatible(commonType, expType) {
						c.fileError(typeMismatch(expType, commonType, rhsRang), t.FileID())
					}
				case arity.MaxParams != -1 && i+tup.Len() > arity.MaxParams:
				// Tuple has more parameters than the function accepts
				// TODO: Error
				default:
					for j, item := range tup.Items {
						expType := fn.Params[min(i+j, len(fn.Params)-1)]
						if !Compatible(item, expType) {
							c.fileError(typeMismatch(expType, item, rhsRang), t.FileID())
						}
					}
					i += tup.Len()
				}
			case KindList:
				itemType = As[*List](rhs.Type).Elem
				fallthrough
			case StringType:
				if !isVariadic {
					c.fileError(misplacedListRestError(rhs.Kind(), rest), t.FileID())
				}
				itemType = cmp.Or[Type](itemType, StringType)
				if !Compatible(itemType, expType) {
					c.fileError(typeMismatch(expType, itemType, rhsRang), t.FileID())
				}
			default:
				c.fileError(invalidRestTypeError(rhs.Type, rest.Expression), t.FileID())
			}
			continue
		}
		e := c.checkExpr(arg.Value, t.NewChild().withHint(expType))
		if !Compatible(e.Type, expType) {
			err := typeMismatch(expType, e.Type, arg.Range)
			// If [T] is passed to a parameter ...T, show a hint
			// TODO: Probably also show a hint for tuples and strings
			if list, ok := Underlying(e.Type).(*List); ok &&
				isVariadic && Compatible(list.Elem, expType) {
				hintWithDiff(
					err, "Did you mean to spread this list?",
					klarerrs.AddedString{Pos: arg.Range.End, String: "..."},
				)
			}
			c.fileError(err, t.FileID())
		}
	}
}

func (c *Checker) checkEnumParams(expr *ast.CallExpression, t *Expr) {
}

func (c *Checker) checkTypeCast(
	typ Type, name string, args []*ast.CallParam, firstArg Type, t *Expr) {
}

// Returned parameters are untyped
func (c *Checker) inferCallParams(params []*ast.CallParam, t *Expr) (ps paramSet) {
	for _, arg := range params {
		var typ Type
		if rest, ok := arg.Value.(*ast.RestExpression); ok {
			inner := c.checkExprFrom(rest.Expression, t)
			switch inner.Kind() {
			case KindTuple:
				tup := As[*Tuple](inner.Type)
				ps.params = append(ps.params, tup.Items...)
				continue
			case KindList:
				typ = As[*List](inner.Type).Elem
			case StringType:
				typ = StringType
			default:
				c.fileError(invalidRestTypeError(inner.Type, rest.Expression), t.FileID())
				typ = inner.Type
			}
		} else {
			typ = c.checkExprFrom(arg.Value, t).Type
		}
		if arg.Label != nil {
			if ps.labelled == nil {
				ps.labelled = make(map[string]Type)
			}
			ps.labelled[arg.Label.Name] = typ
		} else {
			ps.params = append(ps.params, typ)
		}
	}
	return
}

func (c *Checker) checkOverloadParams(o *Overload, ps paramSet, params []*ast.CallParam, t *Expr) {
}
