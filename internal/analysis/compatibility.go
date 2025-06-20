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

func (c *Checker) CheckCompatible(
	expected Type, expr ast.Node, ctx context,
) (gotType Type, ok bool) {
	gotType = c.InferType(expr, ctx)
	return gotType, c.IsCompatibleType(expected, gotType)
}

func (c *Checker) CheckSameType(
	left, right ast.Node, op lexer.TokenType, ctx context,
) Type {
	var (
		expType    Type
		got1, got2 Type
		ok1, ok2   bool
		err        = func(exp, got Type, node ast.Node) {
			c.Error(errors.TypeError{
				ErrorCode:    errors.ErrMismatchedOp,
				Range:        node.Base().Range,
				ExpectedType: exp,
				GotType:      got,
				Params: errors.ErrorParams{"operator": op},
			})
		}
	)
	expType, ok1 = c.InferType(left, ctx), true
	got2, ok2 = c.CheckCompatible(got1, right, ctx)
	if !ok1 {
		err(expType, got1, left)
	}
	if !ok2 {
		err(expType, got2, right)
	}
	return expType
}
