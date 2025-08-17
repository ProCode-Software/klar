package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

func IsArithmetic(op lexer.TokenType) bool {
	switch op {
	case lexer.Plus, lexer.Minus, lexer.Asterisk,
		lexer.Slash, lexer.Percent, lexer.Caret:
		return true
	}
	return false
}

func IsLogical(op lexer.TokenType) bool {
	return op == lexer.Not || op == lexer.AndAnd || op == lexer.OrOr
}

func IsRelational(op lexer.TokenType) bool {
	switch op {
	case lexer.EqualEqual, lexer.NotEqual, lexer.GreaterThan, lexer.LessEqualTo,
		lexer.LessThan, lexer.GreaterEqualTo:
		return true
	}
	return false
}

func IsDistributive(op lexer.TokenType) bool {
	return op == lexer.And || op == lexer.Or
}

// IsRelCompType returns true if t is a relationally comparable type. This only applies to
// >, <, >=, and <= operators, because all types in Klar can be compared for equality
func IsRelCompType(t Type) bool {
	switch t {
	case types.Int, types.Float, types.UntypedInt:
		return true
	default:
		return false
	}
}

func (c *Checker) CheckLogicalExpr(
	left, right ast.Node, op lexer.TokenType, ctx context,
) {
	err := func(got Type, node ast.Node) {
		c.Error(errors.TypeMismatch(types.Bool, got, node.GetRange()))
	}
	got1, ok1 := c.CheckCompatibleExpr(types.Bool, left, ctx)
	got2, ok2 := c.CheckCompatibleExpr(types.Bool, right, ctx)
	if !ok1 {
		err(got1, left)
	}
	if !ok2 {
		err(got2, right)
	}
}

// Checking process:
// 1. If *, require String * Int (not Int * String) or number
//   - Error if String * negative integer
//
// 2. Require same type
// 3. If +, require String, Int, Float, List, Map, or tuple
// 4. Require number for any other operation
//
// Note: Per IEEE 754, floats can be divided by 0, resulting in infinity:
//
//	n / 0.0 = +Inf
//	-n / 0.0 = -Inf
//	-n / -0.0 = -Inf
//
// Dividing an Int by a 0 literal will raise a type error
func (c *Checker) CheckArithmetic(
	binExp *ast.BinaryExpression, op lexer.TokenType, ctx context,
) Type {
	var (
		leftNode, rightNode = binExp.Left, binExp.Right
		left, right         = c.InferType(leftNode, ctx), c.InferType(rightNode, ctx)
		result              = left
		errCode             = errors.ErrMismatchedOperands
	)
	fmt.Println(left, right)
	fmt.Printf("%T %T\n", left, right)
	compat := func(got, with Type) bool { return c.IsCompatibleType(with, got) }
	is := func(t Type) bool { return compat(left, t) }
	// 1. String * Int
	if op == lexer.Asterisk && compat(left, types.String) {
		if compat(right, types.Int) {
			return types.String // String * Int = String
		}
		goto mismatchedOperands // String * non-Int
	}
	// 2. Check same type
	// TODO: Disregard generic or different list types
	if !compat(right, left) {
		goto mismatchedOperands
	}
	// 3. If +, require String, Int, Float, List, Map, or tuple
	if op == lexer.Plus {
		// any := types.Optional{types.Any}
		switch {
		case is(types.Int), is(types.Float), IsMap(left),
			IsList(left), IsTuple(left):
			return left
		default:
			goto invalidOperation
		}
	}
	// 4. Require number for any other operation
	if is(types.Int) || is(types.Float) {
		return result
	}
	goto invalidOperation

invalidOperation:
	errCode = errors.ErrInvalidOperation
	goto mismatchedOperands
mismatchedOperands:
	err := errors.OperatorTypeError(binExp.GetRange(), left, right, op)
	err.ErrorCode = errCode
	c.Error(err)
	return types.InvalidType
}
