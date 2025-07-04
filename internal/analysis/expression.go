package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

type Expression = ast.Expression

func (c *Checker) CheckBinaryExpr(expr ast.BinaryExpression, ctx context) Type {
	op := expr.Operator
	switch {
	case op == lexer.In:
		// Always returns Bool
		// Int in [Int] | (String | Int) in (String, Int) | K in Map<K, V>
		return types.Bool
	case IsLogical(op):
		c.CheckLogicalExpr(expr.Left, expr.Right, op, ctx)
		return types.Bool
	case IsDistributive(op):
		return c.CheckSameType(expr, op, ctx)
	case IsRelational(op):
		typ := c.CheckSameType(expr, op, ctx)
		if op != lexer.EqualEqual && op != lexer.NotEqual && !IsRelCompType(typ) {
			c.Error(errors.TypeError{
				GotType:   typ,
				ErrorCode: errors.ErrUncomparableTypes,
				Params:    errors.ErrorParams{"operator": op},
				Range:     expr.GetRange(),
			})
		}
		return typ
	case IsArithmetic(op):
		return c.CheckArithmetic(expr, op, ctx)
	}
	return nil
}

func (c *Checker) CheckList(expr ast.ListLiteral, ctx context) Type {
	if len(expr.Items) == 0 {
		// Untyped empty list
		return types.UntypedList
	}
	var (
		length    = len(expr.Items)
		itemTypes = make(map[Type]bool, length)
		typed     = make([]Type, 0, length)
		untyped   = make(map[types.Untyped]bool, length)
	)
	// 1. Infer each item type
	for _, item := range expr.Items {
		typ := c.InferType(item, ctx)
		if typ, ok := typ.(types.Untyped); ok {
			// If untyped, add to untyped group
			untyped[typ] = true
		} else {
			// Otherwise, it is a type of the final list
			typed = append(typed, typ)
			if !itemTypes[typ] {
				itemTypes[typ] = true
			}
		}
	}
	// 2. Check each typed item if it is an Int, Float, Optional, or List
	for _, t := range typed {
		types.WalkUnionOptional(&t, func(t *types.Type) {
			switch *t {
			case types.Int, types.Float:
				delete(untyped, types.UntypedInt)
			default:
				switch (*t).(type) {
				case types.List:
					delete(untyped, types.UntypedList)
				case types.Optional:
					delete(untyped, types.UntypedNil)
				}
			}
		})
	}
	// 3. Make the final union
	// Add remaining untyped types
	for untypedItem := range untyped {
		typed = append(typed, untypedItem)
	}
	var ofType Type = types.Union{typed}
	ofType = types.FlattenUnion(ofType.(types.Union))
	if len(typed) == 1 {
		// If one type, just that type
		ofType = typed[0]
	} else if len(typed) > 3 {
		// If more than 3 different types, just use any
		ofType = types.Any
	}
	return types.List{ofType}
}
