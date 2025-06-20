package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/types"
)

type Expression = ast.Expression

func (c *Checker) CheckBinaryExpr(expr ast.BinaryExpression, ctx context) Type {
	op := expr.Operator
	switch {
	case IsLogical(op):
		//c.CheckSameType(expr.Left, expr.Right, types.Bool, ctx)
		return types.Bool
	case IsDistributive(op), IsRelational(op):
		return c.CheckSameType(expr.Left, expr.Right, op, ctx)
	case IsArithmetic(op):
	}
	return nil
}

func (c *Checker) CheckList(expr ast.ListLiteral, ctx context) Type {
	if len(expr.Items) == 0 {
		// Untyped empty list
		return types.UntypedList
	}
	itemTypes := make(map[Type]bool, len(expr.Items))
	union := make([]Type, 0, len(expr.Items))
	for _, item := range expr.Items {
		typ := c.InferType(item, ctx)
		if !itemTypes[typ] {
			union = append(union, typ)
			itemTypes[typ] = true
		}
	}
	var ofType Type = types.Union{union}
	if len(union) == 1 {
		// If one type, just that type
		ofType = union[0]
	} else if len(union) > 3 {
		// If more than 3 different types, just use any
		ofType = types.Any
	}
	return types.List{ofType}
}
