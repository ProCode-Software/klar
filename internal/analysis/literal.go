package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

func (c *Checker) checkTupleLiteral(expr *ast.TupleLiteral, t *Expr) {
	tup := make(Tuple, len(expr.Values))
	for i, expr := range expr.Values {
		tup[i] = c.checkExpr(expr, newChildExpr(t, 0)).Type
	}
	t.Type = tup
}

func (c *Checker) checkListLiteral(expr *ast.ListLiteral, t *Expr) {
	// Use t's type hint if available
	var hint Type
	if listHint, ok := t.hint.(*List); ok {
		hint = listHint.Elem
	}

	list := &List{hint}
	for _, expr := range expr.Items {
		e := c.checkExpr(expr, newChildExprWithHint(t, list.Elem, 0))
		list.Elem = e.Type
	}

	if list.Elem == nil {
		if (t.mode & infer) == 0 {
			t.Type = Untyped(KindList)
			return
		}
		// No hint and no list items: unknown list type
		err := klarerrs.Node(klarerrs.ErrUntypedEmptyList, expr)
		err.Label = "This list is empty and its type can't be inferred"

		// Suggest hints
		err.Hint("If you're declaring a variable, add a type annotation before ':='.")

		diff2 := klarerrs.NewDiff(
			c.module.ResolveFilePath(t.Context.File),
			klarerrs.AddedString{Position: expr.Range.Start, String: "[T]("},
			klarerrs.AddedString{Position: expr.Range.End, String: ")"},
		)
		err.HintWithDiff(
			"Otherwise, initialize an empty list with a specific type. (Replace 'T' with the intended item type)",
			diff2,
		)

		c.fileError(err, t.Context.File)
		list.Elem = InvalidType
	}
	t.Type = list
}

func (c *Checker) checkNilLiteral(expr *ast.NilLiteral, t *Expr) {
	var hint Type
	if opt, ok := t.hint.(*Optional); ok {
		hint = opt.Elem
	}
	if hint == nil {
		if (t.mode & infer) == 0 {
			t.Type = Untyped(KindOptional)
			return
		}
		// Untyped nil
		err := klarerrs.Node(klarerrs.ErrUntypedNil, expr)
		err.Label = "I don't know what optional type this is"
		c.fileError(err, t.Context.File)
		hint = InvalidType
	}
	t.Type = &Optional{hint}
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
		e := c.checkExpr(interp.Expression, newChildExpr(t, 0))
		_ = e
	}
}
