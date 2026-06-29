package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
)

// TODO: Implement. For now, subjects and bodies are checked
func (c *Checker) checkWhenExpr(expr *ast.WhenExpression, t *Expr) {
	for _, subj := range expr.Subjects {
		c.checkExpr(subj, newChildExpr(t, infer))
	}
	valHint := t.hint
	for i, cs := range expr.Cases {
		bodyCtx := NewContext(t.Context, t.Context.File)
		stmtFlags := allowNextStop
		if i == len(expr.Cases)-1 {
			stmtFlags |= finalWhenCase // Forbid 'next' in the final case
		}
		switch body := cs.Body.(type) {
		case *ast.Block:
			sctx := newChildStmtContext(t.stmtCtx, bodyCtx, stmtFlags)
			c.checkBlock(body.Body, sctx)
		case ast.Statement:
			sctx := newChildStmtContext(t.stmtCtx, bodyCtx, stmtFlags|braceless)
			c.checkStmt(body, sctx)
		case ast.Expression:
			_ = valHint
			/* e := c.checkExpr(body, newChildExprWithHint(t, valHint, 0))
			valHint = e.Type */
		}
	}
}

func (c *Checker) checkStringTypeMatch(tm *ast.StringTypeMatch, t *Expr) {
	typ := c.parseType(tm.Type, t.Context)
	// Allowed as types:
	// - String (redundant, show error)
	// - Int
	// - Float
	// - Bool
	// - List of the types above (not tuples)
	// - Tuple of the types above, except tuples
	// - Optional of all the types above
	//
	// In the future, type T will be allowed if T(String) is a defined initializer
	t.Type = typ // Not needed
}
