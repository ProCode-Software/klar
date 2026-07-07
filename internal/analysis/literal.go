package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

func (c *Checker) checkTupleLiteral(expr *ast.TupleLiteral, t *Expr) {
	tup := make(Tuple, len(expr.Values))
	for i, expr := range expr.Values {
		tup[i] = c.checkExprFrom(expr, t).Type
	}
	t.Type = tup
}

func (c *Checker) checkListLiteral(expr *ast.ListLiteral, t *Expr) {
	if len(expr.Items) == 0 { // Empty list
		t.Type = Untyped(KindList)
		return
	}
	// Use t's type hint if available
	var hint Type
	if t.hint != nil && t.hint.Kind() == KindList {
		hint = Underlying(t.hint).(*List).Elem
	}

	list := &List{hint}
	t.Type = list
	for i, item := range expr.Items {
		e := c.checkExprFrom(item, t)
		prev := list.Elem
		if stop := c.inferLiteral(e, &list.Elem, item, hint, func(err *klarerrs.Error) {
			err.SetParam("kind", "list")
			err.AddHighlight(
				"The previous item has type "+quoteAka(prev), expr.Items[i-1].GetRange(),
			)
		}); stop {
			return
		}
	}
}

func (c *Checker) checkNilLiteral(expr *ast.NilLiteral, t *Expr) {
	switch {
	case t.hint != nil && t.hint.Kind() == KindOptional:
		t.Type = t.hint // Hint is optional
	case t.hint == nil:
		t.Type = Untyped(KindOptional) // No hint
	default:
		// Hint is not optional
		err := klarerrs.Node(klarerrs.ErrNotOptionalType, expr)
		err.Name = quoteAka(t.hint)
		err.Label = "This can't be applied to type " + err.Name
		c.fileError(err, t.Context.File)
		t.Type = t.hint
	}
}

func (c *Checker) checkStringLiteral(expr *ast.StringLiteral, t *Expr) {
	t.Type = StringType
	// Check all interpolations
	for _, frag := range expr.Fragments {
		interp, ok := frag.(*ast.InterpolationFragment)
		if !ok {
			continue
		}
		if tm, ok := interp.Expression.(*ast.StringTypeMatch); ok {
			// TODO: Store a whenContext in *Expr to declare pattern-matched variables
			c.checkStringTypeMatch(tm, t)
			continue
		}
		// TODO: Check that each expression can be cast to String
		// And disallow certain expression nodes (using an exprMode)
		e := c.checkExprFrom(interp.Expression, t)
		_ = e
	}
}

func (c *Checker) checkMapLiteral(expr *ast.MapLiteral, t *Expr) {
	// Untyped empty map
	if len(expr.Entries) == 0 {
		t.Type = Untyped(KindMap)
		return
	}

	mp := &Map{}
	t.Type = mp
	// Use map hint if available
	var hasHint bool
	if t.hint != nil && t.hint.Kind() == KindMap {
		hintMap := Underlying(t.hint).(*Map)
		mp.Key, mp.Value = hintMap.Key, hintMap.Value
		hasHint = true
	}

	for i, entry := range expr.Entries {
		// Value
		var valHint Type
		if hasHint {
			valHint = mp.Key
		}
		v := c.checkExprFrom(entry.Value, t)
		prev := mp.Value
		if stop := c.inferLiteral(v, &mp.Value, entry.Value, valHint, func(err *klarerrs.Error) {
			err.SetParam("kind", "map")
			err.AddHighlight(
				"The previous value has type "+quoteAka(prev),
				expr.Entries[i-1].Value.GetRange(),
			)
		}); stop {
			return
		}

		// Keys
		var keyHint Type
		if hasHint {
			keyHint = mp.Key
		}
		for j, key := range entry.Keys {
			k := c.checkExprFrom(key, t)
			prev := mp.Key
			if stop := c.inferLiteral(k, &mp.Key, key, keyHint, func(err *klarerrs.Error) {
				err.SetParam("kind", "map")
				var prevKey ast.Node
				if j > 0 {
					prevKey = entry.Keys[j-1]
				} else {
					prevEntry := expr.Entries[i-1]
					prevKey = prevEntry.Keys[len(prevEntry.Keys)-1]
				}
				err.AddHighlight(
					"The previous key has type "+quoteAka(prev), prevKey.GetRange(),
				)
			}); stop {
				return
			}
		}
	}
}

func (c *Checker) inferLiteral(e *Expr, inferred *Type,
	node ast.Node, hint Type, onError func(*klarerrs.Error),
) (stop bool) {
	if hint != nil && !Compatible(e.Type, *inferred) {
		err := typeMismatch(*inferred, e.Type, node.GetRange())
		err.Node = node
		if onError != nil {
			onError(err)
		}
		c.fileError(err, e.FileID())
	} else if hint == nil {
		prev := *inferred
		if *inferred = commonTypeOptional(*inferred, e.Type); *inferred == nil {
			// List items must have the same type
			err := typeMismatch(prev, e.Type, node.GetRange())
			err.Code = klarerrs.ErrInvalidCollectionType
			err.Node = node
			if onError != nil {
				onError(err)
			}
			c.fileError(err, e.FileID())
			*inferred = InvalidType
			return true
		}
	}
	return false
}

func (c *Checker) checkRegexLiteral(expr *ast.RegexLiteral, t *Expr) {
	t.Type = RegExType
	// TODO: Check interpolations
}

func (c *Checker) checkEnumLiteral(expr *ast.EnumLiteral, t *Expr) {
	if t.hint != nil && t.hint.Kind() == KindEnum {
		t.Type = t.hint
		return
	}
	t.Type = &UntypedInit{kind: KindEnum, Node: expr}
}
