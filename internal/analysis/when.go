package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type whenSubject struct {
	*Expr
	Node ast.Expression // Can be nil
}

type WhenPattern struct {
	Kind WhenPatternKind
	Vars map[string]WhenVar // Variables unwrapped in the pattern
	*Expr
}

type WhenPatternKind int

const (
	InvalidPattern WhenPatternKind = iota

	LiteralExprPattern // Literal expression
	DefaultPattern     // _
	BinaryPattern      // Binary expression with inferred LHS
	StringPattern      // String with interpolation patterns
	RangePattern       // Range. TODO: Should we remove this and use `in 1...10` instead?
	TypePattern        // Type match
	ListPattern        // List length match
	MapPattern         // TODO
	EnumPattern        // Enum
)

type WhenVar struct {
	Value     ast.Expression
	DeclRange ranges.Range
	Type      Type
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

// TODO: Implement. For now, subjects and bodies are checked
func (c *Checker) checkWhenExpr(expr *ast.WhenExpression, t *Expr) {
	subjects := make([]*Expr, len(expr.Subjects))
	for i, subj := range expr.Subjects {
		subjects[i] = c.checkExprFrom(subj, t) // TODO: Convert to typed
	}
	for caseI, cs := range expr.Cases {
		bodyCtx := NewContext(t.Context, t.Context.File)

		// To ensure variables are declared in all options
		varCounter := make(map[string]int)
		_ = varCounter
		for _, opt := range cs.Options { // Separated by '|'
			for subjI, patExpr := range opt { // Separated by ','
				var ws *whenSubject
				if len(subjects) == 0 {
					// Implicit when (see docs for [*whenSubject.IsImplicitTrue])
					ws = newImplicitTrueSubject(t.Context) // Not bodyCtx
				} else {
					ws = &whenSubject{Expr: subjects[subjI], Node: expr.Subjects[subjI]}
				}
				c.checkWhenPattern(ws, patExpr)
			}
		}

		// As

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

func (pat *WhenPattern) declareVar(
	ident ast.Identifier, typ Type, valNode ast.Expression,
) *klarerrs.Error {
	if ident.IsDiscard() {
		return nil
	} else if pat.Vars == nil {
		pat.Vars = make(map[string]WhenVar)
	} else if existing, ok := pat.Vars[ident.Name]; ok {
		err := klarerrs.Node(klarerrs.ErrRedeclared, ident)
		err.Details = append(err.Details, klarerrs.Detail{
			Range:   existing.DeclRange,
			Message: klarerrs.Quote(ident.Name) + " was originally declared here",
		})
		err.Label = klarerrs.Quote(ident.Name) + " was already declared in this pattern"
		err.Name = ident.Name
		return err
	}
	pat.Vars[ident.Name] = WhenVar{
		DeclRange: ident.Range(),
		Type:      typ,
		Value:     valNode,
	}
	return nil
}

func (c *Checker) checkWhenPattern(ws *whenSubject, expr ast.Expression) *WhenPattern {
	pat := &WhenPattern{
		Kind: LiteralExprPattern,
		Expr: ws.Expr.NewChild(patternMatch).withHint(ws.Type),
	}
	pat.Type = ws.Type
	// If the when has no subjects, each case must evaluate to Bool. Pattern
	// matching will be disabled.
	if ws.IsImplicitTrue() {
		c.checkImplicitTrueWhenPattern(expr, pat)
		return pat
	}
	// Some cases such as relational pattern matching will be stored with type
	// Bool. In those cases, we shouldn't check for compatibility.
	checkCompat := true

	switch expr := expr.(type) {
	case *ast.Discard:
		pat.Kind, pat.Type = DefaultPattern, ws.Type
		checkCompat = false
	case *ast.BinaryExpression:
		if expr.Left != nil {
			// Literal expression pattern
			c.checkExpr(expr, pat.Expr)
			pat.Kind = LiteralExprPattern
			break
		}
		pat.Kind = BinaryPattern
		checkCompat = false
		rhs := c.checkExprFrom(expr.Right, pat.Expr)
		pat.Type = c.checkBinaryOperation(
			expr.Operator, ws.Type, rhs.Type,
			ws.Node, expr.Right, expr, ws.FileID(),
		) // Bool, because this is parsed for relational operators only
	case *ast.RelationalExpression:
		if expr.Expressions[0] != nil {
			break
		}
		pat.Kind = BinaryPattern
		checkCompat = false
		// Just to pass to [Checker.checkRelationalExpr]
		expr2 := new(*expr)
		expr2.Expressions[0] = ws.Node
		c.checkRelationalExpr(expr2, pat.Expr)
	case *ast.StringLiteral:
		c.checkStringWhenPattern(expr, ws, pat)
	case *ast.RangeExpression:
		switch ws.Type.Kind() {
		default:
			fallthrough // Type mismatch. Will be reported later
		case KindList:
			// Normal list equality check
			// 	when [1, 2, 3] { 1...3 -> ... }
			pat.Kind = LiteralExprPattern
			c.checkExpr(expr, pat.Expr)
		case IntType, FloatType, StringType: // when 'a' { 'a'...'z' -> ... }
			pat.Kind = RangePattern
			// Checking expr will yield a list because it's a range, but
			// the subject isn't a list
			c.checkRangeExpr(expr, pat.Expr.withHint(&List{ws.Type}))
			// Set the pattern's type to the Int/Float/String so we can
			// check for compatibility later.
			if list, ok := Underlying(pat.Type).(*List); ok {
				pat.Type = list.Elem
			}
		}
	case *ast.CallExpression:
	// Type, enum, or literal pattern
	case *ast.StructDotInit:
	case *ast.ListLiteral:
		c.checkListWhenPattern(ws, expr, pat)
	case *ast.MapLiteral:
	case *ast.EnumLiteral:
	case *ast.RestExpression:
		// Could be used in a string or range pattern
	case *ast.Symbol:
		// Could be a type
		obj := ws.Context.LookupRecursive(expr.Identifier)
		switch {
		case obj == nil:
			c.fileError(klarerrs.Undefined(expr.Identifier, expr.Range), ws.FileID())
			pat.Kind, pat.Type = InvalidPattern, InvalidType
			return pat
		case obj.IsTypeName():
			pat.Kind, pat.Type = TypePattern, obj
			checkCompat = false
			// This pattern is only allowed on non-concrete types.
		default:
			pat.Kind, pat.Type = LiteralExprPattern, obj
		}
	default: // Including parentheses
		c.checkExpr(expr, pat.Expr)
	}
	if checkCompat && !Compatible(pat.Type, ws.Type) {
		err := typeMismatch(ws.Type, pat.Type, expr.GetRange())
		err.AddHighlight("The subject has type "+quoteAka(ws.Type), ws.Node.GetRange())
		c.fileError(err, pat.FileID())
		pat.Kind = InvalidPattern
		return pat
	}
	return pat
}

// For 'when' expressions without subjects, all cases must be boolean. Optionals
// are allowed too.
//
//	when {
//	  !optional -> {}
//	  x == 1 -> {}
//	}
func (c *Checker) checkImplicitTrueWhenPattern(expr ast.Expression, pat *WhenPattern) {
	switch expr := expr.(type) {
	case *ast.Discard:
		pat.Kind = DefaultPattern
		return
	case *ast.BinaryExpression:
		if expr.Left == nil {
			// TODO: Report a different error
			break
		}
		c.checkExpr(expr, pat.Expr)
	case *ast.RelationalExpression:
		if expr.Expressions[0] == nil {
			// TODO: Report a different error
			break
		}
		c.checkExpr(expr, pat.Expr)
	default:
		c.checkExpr(expr, pat.Expr)
	}
	pat.Kind = LiteralExprPattern
	if !Compatible(pat.Type, BoolType) && pat.Type.Kind() != KindOptional {
		err := klarerrs.Node(klarerrs.ErrWhenTrueMismatch, expr)
		err.Label = "This should have type Bool"
		c.fileError(err, pat.FileID())
	}
}

func (c *Checker) checkStringWhenPattern(lit *ast.StringLiteral, ws *whenSubject, pat *WhenPattern) {
	pat.Type = StringType
	var hasDiscard bool
	for _, frag := range lit.Fragments {
		interp, ok := frag.(*ast.InterpolationFragment)
		if !ok {
			continue
		}
		// Declare unwrapped variables from interpolations
		var name ast.Identifier
		var varType Type
		switch inner := interp.Expression.(type) {
		case *ast.Symbol:
			name, varType = inner.ToIdentifier(), StringType
		case *ast.Discard: // No variables to declare
			hasDiscard = true
		case *ast.StringTypeMatch:
			t := ws.NewChild()
			c.checkStringTypeMatch(inner, t) // *Expr value isn't needed
			name, varType = inner.Name, t.Type
		default:
			// Normal interpolation
			c.checkStringInterpolation(inner, ws.NewChild())
		}
		if !name.IsZero() {
			if err := pat.declareVar(name, varType, interp.Expression); err != nil {
				c.fileError(err, ws.FileID())
			}
		}
	}
	// The pattern will only be set to [StringPattern] if there is at
	// least 1 thing being matched. String with no variables or discards
	// declared will be just a literal expression.
	if len(pat.Vars) > 0 || hasDiscard {
		pat.Kind = StringPattern
	} else {
		pat.Kind = LiteralExprPattern
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
			for _, item := range As[*Tuple](typ).Items {
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

func (c *Checker) checkListWhenPattern(ws *whenSubject, list *ast.ListLiteral, pat *WhenPattern) {
}
