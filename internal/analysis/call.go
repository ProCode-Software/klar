package analysis

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

func (c *Checker) checkCallExpr(expr *ast.CallExpression, t *Expr) {
	lhs := c.checkExpr(expr.Callee, t.NewChild(callLHS))
	if lhs.Type.Kind() == InvalidType {
		t.Type = InvalidType
		return
	}
	canCall := true
	switch fn := UnderlyingTypeName(lhs.Type).(type) {
	case *Overload, *Lambda, *TypeName, *EnumFunction:
	case *Function:
		if isTODO(fn) {
			t.mode |= todoExpr
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
	}
	if !canCall {
		// Not a function (or initializer)
		err := klarerrs.Node(klarerrs.ErrNotAFunction, expr.Callee)
		// If the user tries to call an enum item that doesn't take parameters,
		// show a different error
		if ei, ok := Underlying(lhs.Type).(*EnumRef); ok {
			err.Code = klarerrs.ErrEnumItemNoParams
			err.Label = fmt.Sprintf("Can't pass parameters to %s.%s", ei.Enum.Name, ei.name)
			err.Name = ei.name
		} else {
			typ := quoteAka(lhs.Type)
			err.Label = "This callee has type " + typ + " and can't be called"
			err.Name = typ
		}
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	}
	c.checkCallArgs(lhs.Type, expr, t)
}

func (c *Checker) checkCallArgs(lhs Type, expr *ast.CallExpression, t *Expr) {
	oldLHS := lhs
	switch fn := Underlying(lhs).(type) {
	case *Struct:
		t.Type = lhs // TODO
	case *Enum:
		t.Type = lhs
	case *Interface:
		t.Type = lhs
	case *Tag:
		t.Type = lhs
	case *EnumRef:
		t.Type = lhs
	case *Lambda:
		c.checkLambdaParams(fn, expr, t)
	case *Function:
		t.Type = fn.Return
	case *Overload:
		t.Type = fn.Return
	default:
		panic(fmt.Sprintf(
			"checkCallArgs: unhandled LHS type after being handled by checkCallExpr: %T",
			oldLHS,
		))
	}
}

func (c *Checker) checkArity(call *ast.CallExpression, arity Arity, got int, fid FileID) bool {
	if arity.InRange(got) {
		return true
	}
	err := klarerrs.Slice(klarerrs.ErrWrongParamCount, call.Args)
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

func (c *Checker) checkLambdaParams(fn *Lambda, expr *ast.CallExpression, t *Expr) {
	t.Type = fn.Return
	arity := fn.Arity()
	if !c.checkArity(expr, arity, len(expr.Args), t.FileID()) {
		return
	}
	for i := 0; i < len(expr.Args); i++ {
		var (
			arg        = expr.Args[i]
			isVariadic = i > len(fn.Params)
			expType    = fn.Params[min(i, len(fn.Params)-1)]
		)
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
				case isVariadic:
					commonType, err := canSpreadTuple(tup)
					if err != nil {
						err.Range = rhsRang
						c.fileError(err, t.FileID())
						break
					}
					if !Compatible(commonType, expType) {
						c.fileError(typeMismatch(expType, commonType, rhsRang), t.FileID())
					}
				case len(tup.Items) == 0:
					c.fileError(
						klarerrs.Node(klarerrs.ErrSpreadEmptyTuple, rest.Expression).
							WithLabel("This tuple is empty"),
						t.FileID(),
					)
				case arity.MaxParams != -1 && i+len(tup.Items) > arity.MaxParams:
				// Tuple has more parameters than the function accepts

				default:
					for j, item := range tup.Items {
						expType := fn.Params[min(i+j, len(fn.Params)-1)]
						if !Compatible(item, expType) {
							c.fileError(typeMismatch(expType, item, rhsRang), t.FileID())
						}
					}
					i += len(tup.Items)
				}
			case KindList:
				itemType = As[*List](rhs.Type).Elem
				fallthrough
			case StringType:
				if !isVariadic {
					c.fileError(dynamicRestError(rhs.Kind(), rest), t.FileID())
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
					klarerrs.AddedString{Position: arg.Range.End, String: "..."},
				)
			}
			c.fileError(err, t.FileID())
		}
	}
}

func (c *Checker) checkStructDotInitParams(expr *ast.StructDotInit, t *Expr) {
}

func (c *Checker) checkEnumParams(expr *ast.CallExpression, t *Expr) {
}
