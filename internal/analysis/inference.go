package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) InferType(expr ast.Node, ctx context) Type {
	switch expr := expr.(type) {
	case *ast.IntegerLiteral:
		return types.UntypedInt
	case *ast.FloatLiteral:
		return types.Float
	case *ast.StringLiteral:
		return types.String
	case *ast.BooleanLiteral:
		return types.Bool
	case *ast.RegexLiteral:
		return types.Regex
	case *ast.Symbol:
		decl, found := ctx.Resolve(expr.Identifier)
		if !found {
			c.ErrUndefinedVar(expr.Identifier, expr.GetRange(), ctx)
			return types.InvalidType
		}
		return types.Ref{Name: expr.Identifier, Value: &decl.Type}
	case *ast.EnumLiteral:
		return types.UntypedEnum{Name: expr.Name}
	case *ast.NilLiteral:
		return types.UntypedNil
	case *ast.BadExpression:
		return types.InvalidType
	case *ast.ListLiteral:
		return c.CheckList(expr, ctx)
	case *ast.TupleLiteral:
		items := make([]Type, len(expr.Values))
		for i, item := range expr.Values {
			itemType := c.InferType(item, ctx)
			items[i] = itemType
		}
		return types.Tuple{items}
	case *ast.MapLiteral:
		return types.Map{} // TODO
	case *ast.BinaryExpression:
		return c.CheckBinaryExpr(expr, ctx)
	}
	panic(fmt.Sprintf("cannot infer type of %T: not implemented", expr))
}

/* func (c *Checker) CreateUnion() types.Union {

} */

func (c *Checker) ToTyped(t, hint Type) Type {
	var hasFloat bool
	var lastUnion *Type
	var unionItems []*Type
	check := func() {
		for _, typePtr := range unionItems {
			if *typePtr != types.UntypedInt {
				continue
			}
			if hasFloat {
				*typePtr = types.Float
			} else {
				*typePtr = types.Int
			}
		}
		unionItems = unionItems[:0]
		lastUnion = nil
		hasFloat = false
	}
	t = types.Walk(t, func(t, parent *Type) {
		if parent == nil || (lastUnion != nil && parent != lastUnion) {
			return
		}
		if _, isUnion := (*parent).(types.Union); !isUnion {
			check()
			return
		}
		lastUnion = parent
		if *t == types.Float {
			hasFloat = true
		}
		unionItems = append(unionItems, t)
	})
	check()
	return t
}
