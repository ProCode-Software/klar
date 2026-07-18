package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/target"
)

func (c *Checker) checkTupleLiteral(expr *ast.TupleLiteral, t *Expr) {
	tup := &Tuple{make([]Type, len(expr.Values))}
	for i, expr := range expr.Values {
		tup.Items[i] = c.checkExprFrom(expr, t).Type
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
		c.inferCollection(e, &list.Elem, item, hint, func(err *klarerrs.Error) {
			if err.Code == klarerrs.ErrTypeMismatch {
				return
			}
			err.SetParam("kind", "list")
			err.AddHighlight(
				"The previous item has type "+quoteAka(prev), expr.Items[i-1].GetRange(),
			)
		})
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
		interp, ok := frag.(ast.InterpolationFragment)
		if !ok {
			continue
		}
		// TODO: Check that each expression can be cast to String
		// And disallow certain expression nodes (using an exprMode)
		e := c.checkExprFrom(interp.Expression, t, stringInterpolation)
		c.checkStringInterpolation(interp.Expression, e)
	}
}

func (c *Checker) checkStringInterpolation(node ast.Expression, e *Expr) {
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
		v := c.checkExpr(entry.Value, t.NewChild().withHint(valHint))
		prev := mp.Value
		c.inferCollection(v, &mp.Value, entry.Value, valHint, func(err *klarerrs.Error) {
			if err.Code == klarerrs.ErrTypeMismatch {
				return
			}
			err.SetParam("kind", "map")
			err.AddHighlight(
				"The previous value has type "+quoteAka(prev),
				expr.Entries[i-1].Value.GetRange(),
			)
		})

		// Keys
		var keyHint Type
		if hasHint {
			keyHint = mp.Key
		}
		for j, key := range entry.Keys {
			k := t.NewChild()
			if _, ok := key.(*ast.Symbol); ok {
				k.Type = StringType
			} else {
				c.checkExpr(key, k.withHint(keyHint))
			}
			prev := mp.Key
			c.inferCollection(k, &mp.Key, key, keyHint, func(err *klarerrs.Error) {
				if err.Code == klarerrs.ErrTypeMismatch {
					return
				}
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
			})
		}
	}
}

func (c *Checker) inferCollection(e *Expr, inferred *Type,
	node ast.Node, hint Type, onError func(*klarerrs.Error),
) {
	if hint != nil && !Compatible(e.Type, *inferred) {
		err := typeMismatch(*inferred, e.Type, node.GetRange())
		err.Node = node
		if onError != nil {
			onError(err)
		}
		c.fileError(err, e.FileID())
	} else if hint == nil && *inferred != InvalidType {
		prev := *inferred
		if *inferred = commonTypeOptional(*inferred, e.Type); *inferred == nil {
			// If we have 'none' and then a non-optional T (or vice versa), allow
			// inferring as an optional T?.
			if prev != nil && prev.Kind() != e.Type.Kind() {
				if Underlying(e.Type) == Untyped(KindOptional) {
					*inferred = &Optional{prev}
					return
				} else if Underlying(prev) == Untyped(KindOptional) {
					*inferred = &Optional{e.Type}
					return
				}
			}

			// List items must have the same type
			err := typeMismatch(prev, e.Type, node.GetRange())
			err.Code = klarerrs.ErrInvalidCollectionType
			err.Node = node
			if onError != nil {
				onError(err)
			}
			c.fileError(err, e.FileID())
			*inferred = InvalidType
		}
	}
}

// TODO: Should we replace struct{}{} with human-friendly names
var RegexFlags = map[target.Target]map[byte]struct{}{
	target.JavaScript: {
		'u': struct{}{}, 'v': struct{}{},
		// TODO
	},
	target.KlarVM: {},
	target.Unknown: { // Shared among all platforms
		'g': struct{}{}, 'i': struct{}{}, 'm': struct{}{},
	},
}

func (c *Checker) checkRegexLiteral(expr *ast.RegexLiteral, t *Expr) {
	t.Type = RegExType

	// Check flags
	flagsStart := expr.GetRange().End.Sub(0, uint32(len(expr.Flags)))
	validFlag := func(flag byte, t target.Target) bool {
		if _, ok := RegexFlags[t][flag]; ok {
			return true
		}
		_, ok := RegexFlags[target.Unknown][flag]
		return ok
	}
	for i, flag := range expr.Flags {
		for targ := range c.Options.NormalizedTargets {
			if validFlag(flag, targ) {
				continue
			}
			err := klarerrs.Position(
				klarerrs.ErrUnknownRegexFlag, flagsStart.Add(0, uint32(i)),
			)
			err.Label = "Invalid regex flag"
			err.Name = string(flag)
			// Only show the target if the program is being compiled for multiple
			if len(c.Options.Targets) > 1 {
				err.SetParam("target", targ.Name())
			}
			// Show a hint if the flag is uppercase and the lowercase flag exists
			if 'A' <= flag && flag <= 'Z' {
				if validFlag(flag-'A'+'a', targ) {
					err.Hint("Flags must be lowercase and are case-sensitive.")
				}
			}
			c.fileError(err, t.FileID())
		}
	}

	// Check interpolations
	for _, frag := range expr.Fragments {
		if _, ok := frag.(ast.TextFragment); ok {
			continue
		}
		interp := frag.(ast.InterpolationFragment)
		c.checkExprFrom(interp.Expression, t)
	}
}

func (c *Checker) checkEnumLiteral(expr *ast.EnumLiteral, t *Expr) {
	if t.hint != nil && t.hint.Kind() == KindEnum {
		t.Type = t.hint
		return
	}
	t.Type = &UntypedInit{kind: KindEnum, Node: expr}
	c.queue(func() {}, false)
}
