package analysis

import (
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

// IsComparableType returns true if t is a relationally comparable type. This only applies to
// >, <, >=, and <= operators, because all types in Klar can be compared for equality
func IsComparableType(t Type) bool {
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
		c.Error(errors.TypeMismatch(types.Bool, got, node.Base().Range))
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
