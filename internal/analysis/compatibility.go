package analysis

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) IsCompatibleType(expType, gotType Type) bool {
	
	switch expType := expType.(type) {
	case types.Union:
		return slices.Contains(expType.Options, gotType)
	case types.List:
	default:
		return expType == gotType
	}
	return false
}

func (c *Checker) ToTyped(typ, hint Type) (Type, errors.KlarError) {
	return nil, nil
}

func (c *Checker) CheckCompatibleExpr(
	expected Type, expr ast.Node, ctx context,
) (gotType Type, ok bool) {
	gotType = c.InferType(expr, ctx)
	return gotType, c.IsCompatibleType(expected, gotType)
}

func (c *Checker) CheckSameType(
	left, right ast.Node, op lexer.TokenType, ctx context,
) Type {
	err := func(exp, got Type, node ast.Node) {
		code := errors.ErrMismatchedOperands
		if IsDistributive(op) {
			code = errors.ErrMismatchedDistrib
		}
		c.Error(errors.TypeError{
			ErrorCode:    code,
			Range:        node.Base().Range,
			ExpectedType: exp,
			GotType:      got,
			Params:       errors.ErrorParams{"operator": op},
		})
	}
	expType := c.InferType(left, ctx)
	got2, ok := c.CheckCompatibleExpr(expType, right, ctx)
	if !ok {
		err(expType, got2, right)
	}
	return expType
}
