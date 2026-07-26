package analysis

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type whenSubject struct {
	*Expr
	Node    ast.Expression // Can be nil
	Options []*WhenPattern
	Default *WhenPattern // Default case. Can be nil
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
	PatternKind WhenPatternKind
	Vars        map[string]WhenVar // Variables unwrapped in the pattern
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

func (pat *WhenPattern) Kind() Kind       { return pat.Type.Kind() }
func (pat *WhenPattern) String() string   { return pat.Type.String() }
func (pat *WhenPattern) Underlying() Type { return pat.Type }

type WhenVar struct {
	Value     ast.Expression
	DeclRange ranges.Range
	Type      Type
}

type patternChecker struct {
	*Checker
	pat  *WhenPattern
	subj *whenSubject
}

func (pc *patternChecker) fid() FileID { return pc.pat.FileID() }

func (c *Checker) checkWhenExpr(expr *ast.WhenExpression, t *Expr) {
	fid := t.FileID()
	subjects := make([]*whenSubject, len(expr.Subjects))
	for i, subj := range expr.Subjects {
		e := c.checkExprFrom(subj, t)
		e.Type = c.toTyped(e.Type, nil, subj, fid)
		subjects[i] = &whenSubject{Expr: e, Node: subj}
	}
	nilChecks := make(map[int]int, len(subjects)) // Subject index: Case index
	for caseI, cs := range expr.Cases {
		bodyCtx := NewContext(t.Context, fid)

		// 1. Check patterns and options for each subject
		c.checkWhenOptions(expr, subjects, caseI, nilChecks, t)

		// 2. Smart casts: If this case matches types (only), redeclare the subject
		// with a union of each pattern's type. Perform them now so 'as' aliases
		// have a specific type, and so guards can use them.
		//
		// TODO: MAJOR LIMITATION - If the subject is an index (e.g. `obj.value`),
		// we can't declare it as a variable. So this smart cast only supports
		// subjects that are variables (e.g. `value`).
		c.performWhenSmartCasts(subjects, nilChecks, caseI, bodyCtx)

		// Ensure all unwrapped variables are declared within all cases for each
		// subject, with the same types. If we have:
		// 	when x {
		//   .(a) | a -> ...
		//  }
		// `a` must be declared, with the same type, across all options (`|`)
		for _, subj := range subjects {
			vars := c.checkCommonPatternVars(subj.Options)
			// Declare the variables into the body's context
			for name, wv := range vars {
				obj := NewObject(name, fid, wv.DeclRange, c.module, nil)
				obj.Flags |= ImplicitVar
				// TODO: Should we add a new [VariableKind] WhenVar for unwrapped variables?
				NewVariable(obj, LocalVar, wv.Type)
				c.declare(bodyCtx, obj)
			}
			// Clear patterns for the next case
			subj.Options = subj.Options[:0]
		}

		// Direct pattern aliases. Declare them now so the guard can use them.
		c.declareWhenPatternAliases(subjects, cs.As, bodyCtx)

		// Guard
		if cs.Guard != nil {
			guard := c.checkExpr(cs.Guard, t.NewChild().withContext(bodyCtx))
			// Guard must be a Bool. Like implicit true patterns, optionals are
			// allowed.
			// TODO: Allow negated !optional in both places (via a mode flag)
			if !Compatible(guard.Type, BoolType) && guard.Kind() != KindOptional {
				c.fileError(
					typeMismatch(BoolType, guard.Type, cs.Guard.GetRange()), fid,
				)
			}
		}

		// Body
		c.checkWhenBody(expr, caseI, bodyCtx, t)
	}
}

func (c *Checker) checkWhenOptions(expr *ast.WhenExpression, subjects []*whenSubject,
	caseI int, nilChecks map[int]int, t *Expr,
) {
	cs := expr.Cases[caseI]
	for _, opt := range cs.Options { // Separated by '|'
		for subjI, patExpr := range opt { // Separated by ','
			var ws *whenSubject
			if len(subjects) == 0 {
				// Implicit `when true` (see docs for [*whenSubject.IsImplicitTrue])
				ws = newImplicitTrueSubject(t.Context) // Not bodyCtx
			} else {
				ws = subjects[subjI]
			}
			pat := c.checkWhenPattern(ws, patExpr)
			ws.Options = append(ws.Options, pat)
			// Record the type of the pattern expression as a [*WhenPattern]
			e := ws.NewChild()
			e.Type = pat
			c.Info.Expressions[patExpr] = e
			// If the pattern is a nil literal, record the nil check
			if _, ok := patExpr.(*ast.NilLiteral); ok {
				nilChecks[subjI] = caseI
			}

			// Ensure there is only 1 default case per subject
			switch {
			case pat.PatternKind != DefaultPattern:
			case ws.Default != nil:
				firstDefault := c.findWhenPattern(expr, ws.Default).GetRange()
				err := klarerrs.Node(klarerrs.ErrMultipleDefault, patExpr)
				err.Label = "A '_' case was already defined"
				err.SetParam("multiSubject", len(subjects) > 1)
				err.AddDetail("The first one was here", "", firstDefault)
				err.Hint("This pattern is unreachable anyways, so it's safe to remove this.")
			default:
				ws.Default = pat // First '_' pattern
			}
		}
	}
}

func (c *Checker) performWhenSmartCasts(
	subjects []*whenSubject, nilChecks map[int]int, caseI int, bodyCtx *Context,
) {
	for subjI, subj := range subjects {
		orig, ok := subj.Type.(*Object)
		if !ok {
			continue
		}
		casted := new(*orig)
		types := make([]Type, len(subj.Options))
		for i, opt := range subj.Options {
			// Another smart cast: If one case checks if a subject is nil,
			// the other cases won't be an optional.
			if nilCase, ok := nilChecks[subjI]; ok && nilCase != caseI &&
				opt.Type.Kind() == KindOptional {
				opt.Type = As[*Optional](opt.Type).Elem
			}
			types[i] = opt.Type
		}
		union := NewUnion(types)
		switch orig := orig.Type.(type) {
		case *Variable:
			NewVariable(casted, orig.VarKind, union)
		case *Constant:
			casted.Type = &Constant{Type: union, Value: orig.Value}
		default: // Unsupported
		}
		// Use [Context.Declare] to avoid reporting redeclared errors
		if orig = bodyCtx.Declare(casted); orig != nil {
			panic(fmt.Sprintf("%q redeclared in body context: %#v", orig.Name, orig))
		}
	}
}

func (c *Checker) declareWhenPatternAliases(subjects []*whenSubject, vars []ast.Identifier, bodyCtx *Context) {
	for i, name := range vars {
		if len(subjects) == 0 {
			// Implicit `when true` can't declare variables because they
			// will always be `true`. TODO: Error
			break
		}
		subj := subjects[i]
		if name.IsDiscard() {
			continue
		}
		obj := NewObject(name.Name, bodyCtx.File, name.Range(), c.module, nil)
		// TODO: Use type of pattern, and ensure the variable will have the
		// same type across options
		NewVariable(obj, LocalVar, subj.Type)
		// Variable may have been declared through a smart cast. If the alias
		// has the same name as the subject, also show a warning because
		// it is redundant. TODO
		if casted := bodyCtx.Declarations[name.Name]; casted != nil &&
			casted.Flags.Has(ImplicitVar) {
			delete(bodyCtx.Declarations, name.Name)
			bodyCtx.sortedDecls = nil
		}
		c.declare(bodyCtx, obj)
	}
}

func (c *Checker) checkWhenBody(expr *ast.WhenExpression, caseI int, bctx *Context, t *Expr) {
	cs := expr.Cases[caseI]
	stmtFlags := allowNextStop
	if caseI == len(expr.Cases)-1 {
		stmtFlags |= finalWhenCase // Forbid 'next' in the final case
	}
	// If this 'when' block is an expression, each body must be an expression.
	// Blocks aren't allowed, and the only statements allowed are control statements
	switch body := cs.Body.(type) {
	case *ast.Block:
		if !t.mode.has(exprStmt) {
			err := klarerrs.Node(klarerrs.ErrBlockInWhenExpr, body)
			err.AddHighlight("This 'when' is being used as an expression", expr.Range)
			err.Label = "This is only allowed in a 'when' statement"
			c.fileError(err, t.FileID())
			// We will still check the body
		}
		sctx := newChildStmtContext(t.stmtCtx, bctx, stmtFlags)
		c.checkBlock(body.Body, sctx)
	case ast.Statement:
		sctx := newChildStmtContext(t.stmtCtx, bctx, stmtFlags|braceless)
		c.checkStmt(body, sctx)
	case ast.Expression:
		bodyExpr := NewExpr(bctx, allowNothingValue).withHint(t.hint)
		// Allow functions that return Nothing to be used as bodies in
		// 'when' statements
		if t.mode.has(exprStmt) {
			bodyExpr.mode |= exprStmt
		}
		bodyExpr.stmtCtx = t.stmtCtx
		c.checkExpr(body, bodyExpr)
		if bodyExpr.Kind() == NothingType {
		} else if !t.mode.has(exprStmt) {
			// When the 'when' is being used as an expression, the bodies must
			// have the same type.
			t.Type = t.hint
			prevBodyType := t.Type
			c.inferCollection(bodyExpr, &t.Type, body, t.hint, func(err *klarerrs.Error) {
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

func (c *Checker) checkWhenPattern(ws *whenSubject, expr ast.Expression) *WhenPattern {
	pat := &WhenPattern{
		PatternKind: LiteralExprPattern,
		Expr:        ws.Expr.NewChild(patternMatch).withHint(ws.Type),
	}
	// If the when has no subjects, each case must evaluate to Bool. Pattern
	// matching will be disabled.
	if ws.IsImplicitTrue() {
		c.checkImplicitTrueWhenPattern(expr, pat)
		return pat
	}
	pc := &patternChecker{Checker: c, pat: pat, subj: ws}

	switch expr := expr.(type) {
	case *ast.Discard:
		pat.PatternKind, pat.Type = DefaultPattern, ws.Type
	case *ast.BinaryExpression:
		if expr.Left != nil {
			// Literal expression pattern
			c.checkExpr(expr, pat.Expr)
			pat.PatternKind = LiteralExprPattern
			break
		}
		rhs := c.checkExpr(expr.Right, pat.Expr.NewChild().withHint(ws.Type))
		// Bool, because this is parsed for relational operators only
		c.checkBinaryOperation(
			expr.Operator, ws.Type, rhs.Type,
			ws.Node, expr.Right, expr, ws.FileID(),
		)
		// pat.Type will be set to the operand's type so variables declared with
		// `as` will be the subject, rather than a Bool that is always `true`.
		pat.PatternKind, pat.Type = BinaryPattern, ws.Type
	case *ast.RelationalExpression:
		if expr.Expressions[0] != nil {
			c.checkExpr(expr, pat.Expr)
			break
		}
		// Just to pass to [Checker.checkRelationalExpr]
		vnode := new(*expr)
		vnode.Expressions[0] = ws.Node
		c.checkRelationalExpr(vnode, pat.Expr)
		// Type is set for same reason as BinaryExpression
		pat.PatternKind, pat.Type = BinaryPattern, ws.Type
	case *ast.StringLiteral:
		pc.checkString(expr)
	case *ast.RangeExpression:
		switch ws.Type.Kind() {
		default:
			fallthrough // Type mismatch. Will be reported later
		case KindList:
			// Normal list equality check
			// 	when [1, 2, 3] { 1...3 -> ... }
			pat.PatternKind = LiteralExprPattern
			c.checkExpr(expr, pat.Expr)
		case IntType, FloatType, StringType: // when 'a' { 'a'...'z' -> ... }
			pat.PatternKind = RangePattern
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
		// Type, enum, or literal pattern. Actual function calls aren't
		// allowed in when patterns
		pc.checkUnwrap(expr)
	case *ast.StructDotInit:
		pc.checkStruct(expr.Params)
	case *ast.ListLiteral:
		pc.checkList(expr)
	case *ast.MapLiteral:
		pc.checkMap(expr)
	case *ast.EnumLiteral:
		// Hint was already set for pat.Expr. This ensures the item exists. If
		// the subject isn't an enum, checkEnumLiteral also handles that and the
		// mismatch will be reported later.
		c.checkEnumLiteral(expr, pat.Expr)
	case *ast.RestExpression:
		// String match
		//   ..."{word} " | "{word}\n"... | ..."{word}"... -> word
		errMustBeString := func(expr ast.Expression) {
			err := klarerrs.Node(klarerrs.ErrStringInSegmentMatch, expr)
			c.fileError(err, ws.FileID())
			pat.PatternKind, pat.Type = InvalidPattern, InvalidType
		}
		switch inner := expr.Expression.(type) {
		case *ast.StringLiteral:
			pc.checkString(inner)
			pat.PatternKind = StringPattern
		case *ast.RestExpression:
			if str, ok := inner.Expression.(*ast.StringLiteral); ok {
				pc.checkString(str)
				pat.PatternKind = StringPattern
			} else {
				errMustBeString(inner.Expression)
			}
		default:
			// TODO: Allow literal expressions that have type String
			errMustBeString(inner)
		}
	case *ast.Symbol:
		// Could be a type
		obj := ws.Context.LookupRecursive(expr.Identifier)
		switch {
		case obj == nil:
			c.fileError(klarerrs.Undefined(expr.Identifier, expr.Range), ws.FileID())
			pat.PatternKind, pat.Type = InvalidPattern, InvalidType
			return pat
		case obj.IsTypeName():
			pat.PatternKind, pat.Type = TypePattern, obj
			// This pattern is only allowed on non-concrete types.
		default:
			pat.PatternKind, pat.Type = LiteralExprPattern, obj
		}
	default: // Including parentheses
		c.checkExpr(expr, pat.Expr)
	}
	if pat.PatternKind != DefaultPattern && pat.Type != InvalidType &&
		!Compatible(pat.Type, ws.Type) {
		err := typeMismatch(ws.Type, pat.Type, expr.GetRange())
		err.AddHighlight("The subject has type "+quoteAka(ws.Type), ws.Node.GetRange())
		c.fileError(err, pat.FileID())
		pat.PatternKind = InvalidPattern
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
		pat.PatternKind = DefaultPattern
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
	pat.PatternKind = LiteralExprPattern
	if !Compatible(pat.Type, BoolType) && pat.Type.Kind() != KindOptional {
		err := klarerrs.Node(klarerrs.ErrWhenTrueMismatch, expr)
		err.Label = "This should have type Bool"
		c.fileError(err, pat.FileID())
	}
}

// checkCommonPatternVars ensures the same variables are declared within the
// provided 'when' patterns, with the same type. The variables with the common
// type of each declaration are returned. Variables not declared in all patterns
// or declared with mismatched types will be in vars with type [InvalidType].
// The range of each returned [WhenVar] will be that of its declaration in the
// first option.
func (c *Checker) checkCommonPatternVars(opts []*WhenPattern) (vars map[string]WhenVar) {
	vars = make(map[string]WhenVar, len(opts[0].Vars))
	varCounter := make(map[string]int, len(vars))
	for _, opt := range opts {
		for name, curr := range opt.Vars {
			varCounter[name]++
			// Skip finding common types of variables where an error was already reported
			if prev, ok := vars[name]; ok && prev.Type != InvalidType {
				// Set the variable's type to the common type. If we have
				// `"{a: String | Int}" | [a, nums...]`, `a` will be `String | Int`
				if prev.Type = CommonType(prev.Type, curr.Type); prev.Type == nil {
					// Same variable declared across patterns with uncommon types
					// TODO: Error
					prev.Type = InvalidType
				}
				vars[name] = prev
			} else {
				vars[name] = curr
			}
		}
	}
	// Ensure the counts of each variable equals the number of options
	for name, count := range varCounter {
		if count == len(opts) {
			continue
		}
		// TODO: Report an error

		// Set the result type to invalid
		decl := vars[name]
		decl.Type = InvalidType
		vars[name] = decl
	}
	return vars
}

func (pat *WhenPattern) declareVar(
	id ast.Identifier, typ Type, valNode ast.Expression,
) *klarerrs.Error {
	if id.IsDiscard() {
		return nil
	} else if pat.Vars == nil {
		pat.Vars = make(map[string]WhenVar)
	} else if existing, ok := pat.Vars[id.Name]; ok {
		err := klarerrs.Node(klarerrs.ErrRedeclared, id)
		err.Details = append(err.Details, klarerrs.Detail{
			Range:   existing.DeclRange,
			Message: klarerrs.Quote(id.Name) + " was originally declared here",
		})
		err.Label = klarerrs.Quote(id.Name) + " was already declared in this pattern"
		err.Name = id.Name
		return err
	}
	pat.Vars[id.Name] = WhenVar{
		DeclRange: id.Range(),
		Type:      typ,
		Value:     valNode,
	}
	return nil
}

func (c *Checker) findWhenPattern(
	when *ast.WhenExpression, target *WhenPattern,
) ast.Expression {
	for _, cs := range when.Cases {
		for _, opts := range cs.Options {
			for _, pat := range opts {
				if c.Info.Expressions[pat].Type == target {
					return pat
				}
			}
		}
	}
	return nil
}

// Allows sub-options and as-declarations
func (pc *patternChecker) checkExprAdvanced(expr ast.Expression) *Expr {
	t := pc.pat.NewChild()
	switch expr := expr.(type) {
	case *ast.SubOptions:
		t.Type = pc.checkOptions(expr)
		pc.Info.Expressions[expr] = t
	case *ast.AsExpression:
		t = pc.checkExprAdvanced(expr.Expression) // Could be sub-options
		if err := pc.pat.declareVar(expr.Name, t.Type, expr.Expression); err != nil {
			pc.fileError(err, pc.fid())
		}
		pc.Info.Expressions[expr] = t
	default:
		pc.checkExpr(expr, t)
	}
	return t
}

func (pc *patternChecker) checkOptions(opts *ast.SubOptions) Type {
	types := make([]Type, 0, len(opts.Options))
	for _, opt := range opts.Options {
		types = append(types, pc.checkExprFrom(opt, pc.pat.Expr).Type)
	}
	return NewUnion(types)
}

func (pc *patternChecker) checkString(lit *ast.StringLiteral) {
	pc.pat.Type = StringType
	var hasDiscard bool
	for _, frag := range lit.Fragments {
		interp, ok := frag.(ast.InterpolationFragment)
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
			continue
		case *ast.StringTypeMatch:
			name, varType = inner.Name, pc.checkStringTypeMatch(inner)
		case *ast.SubOptions:
			un := pc.checkOptions(inner)
			// TODO: Check that all options can be converted to String
			if !Compatible(un, StringType) {
				pc.fileError(typeMismatch(StringType, un, inner.Range), pc.fid())
			}
			continue
		default:
			// Normal interpolation
			pc.checkStringInterpolation(inner, pc.subj.NewChild())
			continue
		}
		if !name.IsDiscard() {
			if err := pc.pat.declareVar(name, varType, interp.Expression); err != nil {
				pc.fileError(err, pc.subj.FileID())
			}
		}
	}
	// The pattern will only be set to [StringPattern] if there is at
	// least 1 thing being matched. String with no variables or discards
	// declared will be just a literal expression.
	if len(pc.pat.Vars) > 0 || hasDiscard {
		pc.pat.PatternKind = StringPattern
	} else {
		pc.pat.PatternKind = LiteralExprPattern
	}
}

func (pc *patternChecker) checkList(list *ast.ListLiteral) {
	pc.pat.PatternKind = ListPattern
	var hint Type
	switch {
	case len(list.Items) == 0 && pc.subj.Kind() == KindList:
		pc.pat.Type = pc.subj.Type
		return
	case len(list.Items) == 0:
		pc.pat.Type = Untyped(KindList)
		return
	case pc.subj.Kind() == KindList:
		hint = As[*List](pc.subj.Type).Elem
	}
	listType := &List{hint}
	pc.pat.Type = listType
	// We can't declare variables as soon as we see them, because the list type may
	// not be fully inferred. Items in vars are either symbols or symbol rests.
	var vars []ast.Expression

	for i, item := range list.Items {
		var e *Expr
		switch item := item.(type) {
		case *ast.RestExpression:
			switch item.Expression.(type) {
			case *ast.Symbol: // [items...]
				vars = append(vars, item)
				continue
			case nil: // [x, y, ...]
				continue
			case *ast.Discard:
				panic("'..._' or '_...' rest not rejected")
			default:
				e = pc.pat.NewChild()
				if ok := pc.checkListRest(item, e); !ok {
					continue
				}
			}
		case *ast.Symbol:
			vars = append(vars, item)
			continue
		case *ast.Discard:
			continue
		default:
			// TODO: Redeclared errors will be raised out of order. If a symbol
			// is declared first, then an as-expression with the same name, the error
			// will be on the symbol even though it was declared first.
			e = pc.checkExprAdvanced(item)
		}
		prev := listType.Elem
		// Same as [Checker.checkListLiteral]
		pc.inferCollection(e, &listType.Elem, item, hint, func(err *klarerrs.Error) {
			if err.Code == klarerrs.ErrTypeMismatch {
				return
			}
			err.SetParam("kind", "list")
			err.AddHighlight(
				"The previous item has type "+quoteAka(prev), list.Items[i-1].GetRange(),
			)
		})
	}
	for _, vr := range vars {
		var err *klarerrs.Error
		switch vr := vr.(type) {
		case *ast.Symbol:
			err = pc.pat.declareVar(vr.ToIdentifier(), listType.Elem, vr)
		case *ast.RestExpression:
			err = pc.pat.declareVar(
				vr.Expression.(*ast.Symbol).ToIdentifier(),
				listType, vr,
			)
		}
		if err != nil {
			pc.fileError(err, pc.fid())
		}
	}
}

func (pc *patternChecker) checkMap(mp *ast.MapLiteral) {
	pc.pat.PatternKind = MapPattern
	pc.pat.Type = InvalidType // TODO
}

func (pc *patternChecker) checkUnwrap(expr *ast.CallExpression) {
	switch lhs := expr.Callee.(type) {
	case *ast.EnumLiteral:
		pc.checkEnum(expr)
	case *ast.IndexExpression:
		// May be referencing an enum when matching on an interface. TODO
		pc.pat.Type = InvalidType
	case *ast.Symbol:
		pc.pat.PatternKind = TypePattern
		pc.checkSymbolExpr(lhs, true, pc.pat.Expr)
		kind := pc.pat.Expr.Kind()
		switch {
		case !isTypeName(pc.pat.Type):
			// Function call
			pc.fileError(
				klarerrs.Node(klarerrs.ErrNotAllowedInWhen, expr).
					SetParam("kind", "a function call").SetParam("location", "pattern"),
				pc.subj.FileID(),
			)
			pc.pat.PatternKind = InvalidPattern
		case kind == KindStruct:
			pc.checkStruct(expr.Args)
		case kind == KindInterface:
		//  Interface(field1, field2)
		// TODO: Should this be allowed? Unlike the other LHS types, there's no
		// valid syntax outside of when patterns.
		case kind == ErrorType:
		default:
			// Type can't be initialized (though it may be outside of a when pattern)
			pc.pat.PatternKind = InvalidPattern
			// Don't set the type to InvalidType so we can still report
			// incompatibility errors
		}
	default:
		// Syntactically a function call
		pc.fileError(
			klarerrs.Node(klarerrs.ErrNotAllowedInWhen, expr).
				SetParam("kind", "a function call").SetParam("location", "pattern"),
			pc.subj.FileID(),
		)
	}
}

func (pc *patternChecker) checkEnum(expr *ast.CallExpression) {
	pc.pat.PatternKind = EnumPattern
	pc.checkEnumLiteral(expr.Callee.(*ast.EnumLiteral), pc.pat.Expr)
	e, ok := pc.pat.Type.(*EnumRef)
	if !ok {
		return // Type mismatch. Will be reported later
	}
	if e.Called {
		// User is unwrapping an enum that doesn't have parameters
		err := klarerrs.Node(klarerrs.ErrEnumItemNoParams, expr)
		err.Label = fmt.Sprintf(
			"Can't pass parameters to %s.%s", e.Enum.Name, e.Name,
		)
		err.Name = e.Name
		return
	}
	var hadLabelled bool
	for i, param := range expr.Args {
		if dot, ok := param.Value.(*ast.RestExpression); ok && dot.Expression == nil {
			// Skip: .withParams3(5, ...) | .withParams4(5, ..., 3, 4)
			continue // TODO
		}
		var actualType Type
		if param.Label != nil {
			// Labelled: .withParams(a: 5) | .withParams(:b)
			hadLabelled = true
			if param.Label.IsDiscard() {
				continue // .withParams(:_)
			}
			if actualType = e.ParamByName(param.Label.Name); actualType == nil {
				err := klarerrs.Node(klarerrs.ErrParamLabelUndefined, param.Label)
				err.Desc = "These params are defined: " + strings.Join(
					slices.Sorted(maps.Keys(e.paramMap)), ", ",
				)
				pc.fileError(err, pc.fid())
				actualType = InvalidType
			}
			// withParams(:a)
			if param.Shorthand {
				if err := pc.pat.declareVar(*param.Label, actualType, param.Value); err != nil {
					pc.fileError(err, pc.fid())
				}
				continue
			}
		} else if sym, ok := param.Value.(*ast.Symbol); hadLabelled &&
			(!ok || sym.Identifier != "_") {
			// Discarded positional params are allowed anywhere
			err := klarerrs.Node(klarerrs.ErrLabelledParamLast, expr)
			err.Label = "Labelled parameters must go after positional parameters"
			pc.fileError(err, pc.fid())
		}
		// Positional param
		if actualType == nil {
			if i >= len(e.Params) {
				// TODO: Error
				break
			}
			actualType = e.Params[i]
		}

		switch expr := param.Value.(type) {
		case *ast.Symbol:
			// .withParams(x, _)
			if err := pc.pat.declareVar(
				expr.ToIdentifier(), actualType, param.Value,
			); err != nil {
				pc.fileError(err, pc.fid())
			}
		case *ast.Discard: // .withParams(_) - Nothing to do
		default:
			// Has literal value (may or may not be labelled)
			// 	.withParams(5) | .withParams(a: 5)
			if val := pc.checkExprAdvanced(expr).Type; !Compatible(val, actualType) {
				pc.fileError(
					typeMismatch(actualType, val, expr.GetRange()), pc.fid(),
				)
			}
		}
	}
}

func (pc *patternChecker) checkStruct(params []*ast.CallParam) {
}

func (pc *patternChecker) checkStringTypeMatch(tm *ast.StringTypeMatch) Type {
	fid := pc.subj.FileID()
	typ := pc.parseType(tm.Type, pc.subj.Context)
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
				pc.fileError(err, fid)
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
			pc.fileError(err, fid)
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
		pc.fileError(err, fid)
	}
	return typ
}
