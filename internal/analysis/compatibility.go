package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

func resolveRefs(t Type) Type {
	if ref, ok := t.(types.Ref); ok {
		return *ref.Value
	}
	return t
}

func (c *Checker) IsCompatibleType(exp, got Type) bool {
	// Always true if expected type is optional
	switch got {
	case types.UntypedNil:
		switch exp := exp.(type) {
		case types.Optional:
			return true
		case types.Union:
			for _, opt := range exp.Options {
				if c.IsCompatibleType(opt, got) {
					return true
				}
			}
		}
	case types.UntypedList:
		if _, ok := exp.(types.List); ok {
			return true
		}
	}
	got = c.ToTyped(got, exp)
	switch exp := exp.(type) {
	case types.Union:
		for _, opt := range exp.Options {
			if c.IsCompatibleType(opt, got) {
				return true
			}
		}
	case types.List:
		if got, ok := got.(types.List); ok {
			return c.IsCompatibleType(exp.Of, got.Of)
		}
	case types.Optional:
		return exp == got || c.IsCompatibleType(exp.Underlying, got)
	case types.Tuple:
		if got, ok := got.(types.Tuple); ok {
			if len(got.Items) != len(exp.Items) {
				return false
			}
			for i, gotItem := range got.Items {
				if !c.IsCompatibleType(exp.Items[i], gotItem) {
					return false
				}
			}
			return true
		}
	case types.Untyped:
		switch exp {
		case types.UntypedInt:
			return c.IsCompatibleType(types.Int, got) ||
				c.IsCompatibleType(types.Float, got)
		case types.UntypedList:
			_, ok := got.(types.List)
			return ok
		}
	default:
		return exp == got
	}
	return false
}

func IsList(t Type) bool {
	_, ok := resolveRefs(t).(types.List)
	return ok
}

func IsMap(t Type) bool {
	_, ok := resolveRefs(t).(types.Map)
	return ok
}

func IsTuple(t Type) bool {
	_, ok := resolveRefs(t).(types.Tuple)
	return ok
}

func (c *Checker) CheckCompatibleExpr(
	expected Type, expr ast.Node, ctx context,
) (gotType Type, ok bool) {
	gotType = c.InferType(expr, ctx)
	return gotType, c.IsCompatibleType(expected, gotType)
}

func (c *Checker) CheckSameType(
	expr ast.BinaryExpression, op lexer.TokenType, ctx context,
) Type {
	var (
		left, right = expr.Left, expr.Right
		expType     = c.InferType(left, ctx)
		got, ok     = c.CheckCompatibleExpr(expType, right, ctx)
	)
	if !ok {
		code := errors.ErrMismatchedOperands
		if IsDistributive(op) {
			code = errors.ErrMismatchedDistrib
		}
		c.Error(errors.TypeError{
			ErrorCode:    code,
			Range:        expr.GetRange(),
			ExpectedType: expType,
			GotType:      got,
			Params:       errors.ErrorParams{"operator": op},
		})
	}
	return expType
}
