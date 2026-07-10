package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// TODO: Implement. For now, subjects and bodies are checked
func (c *Checker) checkWhenExpr(expr *ast.WhenExpression, t *Expr) {
	subjects := make([]*Expr, len(expr.Subjects))
	for i, subj := range expr.Subjects {
		subjects[i] = c.checkExprFrom(subj, t) // TODO: Convert to typed
	}
	for caseI, cs := range expr.Cases {
		bodyCtx := NewContext(t.Context, t.Context.File)

		for _, opt := range cs.Options { // Separated by '|'
			for subjI, patExpr := range opt { // Separated by ','
				var ws *whenSubject
				if len(subjects) == 0 {
					// Implicit when (see docs for [*whenSubject.IsImplicitTrue])
					ws = newImplicitTrueSubject(t.Context) // Not bodyCtx
				} else {
					ws = &whenSubject{
						Expr: subjects[subjI],
						rang: expr.Subjects[subjI].GetRange(),
					}
				}
				c.checkWhenPattern(ws, patExpr)
			}
		}

		// Guard
		if cs.Guard != nil {
			e := t.NewChild()
			t.Context = bodyCtx
			c.checkExpr(cs.Guard, e)
		}

		// Body
		stmtFlags := allowNextStop
		if caseI == len(expr.Cases)-1 {
			stmtFlags |= finalWhenCase // Forbid 'next' in the final case
		}
		// If this 'when' block is an expression, each body must be an expression.
		// Blocks aren't allowed, and the only statements allowed are control statements
		switch body := cs.Body.(type) {
		case *ast.Block:
			if t.mode&exprStmt == 0 {
				err := klarerrs.Node(klarerrs.ErrBlockInWhenExpr, body)
				err.AddHighlight("This 'when' is being used as an expression", expr.Range)
				err.Label = "This is only allowed in a 'when' statement"
				c.fileError(err, t.FileID())
				// We will still check the body
			}
			sctx := newChildStmtContext(t.stmtCtx, bodyCtx, stmtFlags)
			c.checkBlock(body.Body, sctx)
		case ast.Statement:
			sctx := newChildStmtContext(t.stmtCtx, bodyCtx, stmtFlags|braceless)
			c.checkStmt(body, sctx)
		case ast.Expression:
			e := NewExpr(bodyCtx).withHint(t.hint)
			// Allow functions that return Nothing to be used as bodies in
			// 'when' statements
			if t.mode&exprStmt != 0 {
				e.mode |= exprStmt
			}
			e.stmtCtx = t.stmtCtx
			c.checkExpr(body, e)
			// When the 'when' is being used as an expression, the bodies must
			// have the same type.
			if t.mode&exprStmt == 0 {
				prevBodyType := t.Type
				c.inferCollection(e, &t.Type, body, t.hint, func(err *klarerrs.Error) {
					if err.Code == klarerrs.ErrTypeMismatch {
						return
					}
					err.AddHighlight(
						"The previous body expression has type "+quoteAka(prevBodyType),
						expr.Cases[caseI-1].Body.GetRange(),
					)
					err.SetParam("kind", "'when' expression")
				})
			}
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
	var validateType func(Type, bool)
	validateType = func(typ Type, allowTuple bool) {
		switch typ.Kind() {
		case StringType, IntType, FloatType, BoolType:
		case KindTuple:
			if !allowTuple {
				err := klarerrs.Node(klarerrs.ErrNestedTupleStrMatch, tm.Type)
				err.Label = "Type " + quoteAka(typ)
				c.fileError(err, t.FileID())
			}
			for _, item := range As[Tuple](typ) {
				validateType(item, false)
			}
		case KindOptional:
			validateType(As[*Optional](typ).Elem, allowTuple)
		case KindList:
			validateType(As[*List](typ).Elem, false)
		default:
			err := klarerrs.Node(klarerrs.ErrInvalidStrMatchType, tm.Type)
			err.Name = typ.Kind().String()
			err.Label = "Type " + quoteAka(typ)
			c.fileError(err, t.FileID())
		}
	}
	validateType(typ, true)
	if typ.Kind() == StringType {
		err := klarerrs.Node(klarerrs.ErrRedundantStrMatch, tm.Type)
		err.Label = "Useless 'String' type annotation"
		err.HintWithDiff("Remove the type annotation", klarerrs.NewDiff(
			"", klarerrs.DeletedRange{
				// Includes colon
				ranges.Range{tm.Name.Range().End, tm.Type.GetRange().End},
			},
		))
		c.fileError(err, t.FileID())
	}
	t.Type = typ // Not needed
}

type whenSubject struct {
	*Expr
	rang ranges.Range
}

// IsImplicitTrue returns whether s represents an implicit 'true' subject.
//
//	when {
//		str.length -> {}
//		!optional -> {}
//	}
func (s *whenSubject) IsImplicitTrue() bool { return s.Expr.Type == nil }

func newImplicitTrueSubject(ctx *Context) *whenSubject {
	return &whenSubject{Expr: NewExpr(ctx)}
}

type WhenPattern struct {
	Node ast.Expression
	Kind WhenPatternKind
	*Expr
}

type WhenPatternKind int

const (
	_ WhenPatternKind = iota

	LiteralExprPattern // Literal expression
	DefaultPattern     // _
	BinaryPattern      // Binary expression with inferred LHS
	StringPattern      // String with interpolation patterns
	RangePattern       // Rang. TODO: Should we remove this and use `in 1...10` instead?
	TypePattern        // Type match
	ListPattern        // List length match
	MapPattern         // TODO
)

func (c *Checker) checkWhenPattern(subj *whenSubject, expr ast.Expression) {
}
