package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Expr struct {
	Type    Type
	hint    Type
	mode    exprMode // Input mode
	gotMode exprMode // Output mode
	Root    *Expr
	Context *Context
	stmtCtx *stmtContext
}

func NewExpr(ctx *Context, flags ...exprMode) *Expr {
	e := &Expr{Context: ctx, mode: parseFlags(flags)}
	e.Root = e
	return e
}

func (e *Expr) NewChild(flags ...exprMode) *Expr {
	const noInherit = 0
	return &Expr{
		Context: e.Context,
		mode:    (e.mode &^ noInherit) | parseFlags(flags),
		stmtCtx: e.stmtCtx,
		Root:    e.Root,
	}
}

func (sctx *stmtContext) newExpr(flags ...exprMode) *Expr {
	e := &Expr{Context: sctx.ctx, mode: parseFlags(flags), stmtCtx: sctx}
	e.Root = e
	return e
}

// hint can be nil
func (e *Expr) withHint(hint Type) *Expr {
	e.hint = hint
	return e
}

type exprMode uint16

const (
	// Input modes
	typeInit  exprMode = 1 << iota
	constExpr          // Can also be output
	exprStmt
	patternMatch
	indexLHS
	stringInterpolation

	// Output modes
	todoExpr
	intfField
)

func (e *Expr) ConstValue() ConstValue {
	return UnknownConst{}
}

func (e *Expr) Kind() Kind     { return e.Type.Kind() }
func (e *Expr) FileID() FileID { return e.Context.File }

func (c *Checker) checkExprFrom(
	expr ast.Expression, parent *Expr, flags ...exprMode,
) (t *Expr) {
	return c.checkExpr(expr, parent.NewChild(flags...))
}

func (c *Checker) checkExpr(expr ast.Expression, t *Expr) *Expr {
	switch expr := expr.(type) {
	case *ast.BinaryExpression:
		c.checkBinaryExpr(expr, t)
	case *ast.RelationalExpression:
		c.checkRelationalExpr(expr, t)
	case *ast.UnaryExpression:
		c.checkUnaryExpr(expr, t)
	case *ast.NilLiteral:
		// TODO: Use [ConstValue]s
		c.checkNilLiteral(expr, t)
	case *ast.StringLiteral:
		c.checkStringLiteral(expr, t)
	case *ast.IntegerLiteral:
		// All numeric literals can be used as Float, so `3.0 + 5` is valid.
		// TODO: use ConstValue
		if t.hint != nil && t.hint.Kind() == FloatType {
			t.Type = FloatType
		} else {
			t.Type = Untyped(IntType)
		}
	case *ast.FloatLiteral:
		t.Type = FloatType
	case *ast.BooleanLiteral:
		t.Type = BoolType
	case *ast.Symbol:
		c.checkSymbolExpr(expr, false, t)
	case *ast.MapLiteral:
		c.checkMapLiteral(expr, t)
	case *ast.TupleLiteral:
		c.checkTupleLiteral(expr, t)
	case *ast.ListLiteral:
		c.checkListLiteral(expr, t)
	case *ast.IndexExpression:
		c.checkIndexExpr(expr, t)
	case *ast.CallExpression:
		c.checkCallExpr(expr, t)
	case *ast.EnumLiteral:
		c.checkEnumLiteral(expr, t)
	case *ast.WhenExpression:
		c.checkWhenExpr(expr, t)
	case *ast.LambdaExpression:
		c.checkLambdaExpr(expr, t)
	case *ast.RangeExpression:
		c.checkRangeExpr(expr, t)
	case *ast.RestExpression:
		// TODO: Check for rest expressions where they are allowed, so we can
		// report an error if they are reached here.
		// They only are allowed in:
		// - Calls
		// - Tuples
		// - Lists
		// - Maps
		// - List slices
		err := klarerrs.Node(klarerrs.ErrInvalidRestExpr, expr)
		err.Label = "Can't use a rest expression here"
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	case *ast.PipelineExpression:
		c.checkPipelineExpr(expr, t)
	case *ast.BadExpression:
		t.Type = InvalidType
	case *ast.SliceExpression:
		c.checkSliceExpr(expr, t)
	case *ast.ParenExpression:
		c.checkExpr(expr.Expression, t)
	case *ast.RegexLiteral:
		c.checkRegexLiteral(expr, t)
	case *ast.VersionLiteral:
		// These are only parsed within attributes.
		// TODO: Find a way to read these when applying attributes
	case *ast.ListCastExpression:
		c.checkListCastExpr(expr, t)
	case *ast.MapCastExpression:
		c.checkMapCastExpr(expr, t)
	case *ast.ObjectPipeline:
		c.checkObjectPipeline(expr, t)
	case *ast.ForExpression:
	// TODO: Factor logic from [Checker.checkForStmt] to use in checkForExpr
	case *ast.StructDotInit:
		c.checkStructDotInitExpr(expr, t)
	case *ast.GoExpression:
		c.checkGoExpr(expr, t)
	case *ast.AwaitExpression:
		c.checkAwaitExpr(expr, t)
	case *ast.AssertExpression:
		c.checkAssertExpr(expr, t)
	case *ast.TryExpression:
		c.checkTryExpr(expr, t)
	default:
		panic(fmt.Sprintf("unhandled expression node type: %T", expr))
	}
	if t.Type == nil {
		t.Type = InvalidType
	}
	// Ensure a function that returns Nothing isn't being used as a value
	// TODO: Should we move this to function/pipeline/try/await checking?
	if t.Type.Kind() == NothingType && t.mode&exprStmt == 0 {
		err := klarerrs.Node(klarerrs.ErrNothingAsValue, expr)
		err.Label = "This expression returns 'Nothing'"
		c.fileError(err, t.Context.File)
	}
	if nr, ok := t.Type.(*NoReturn); ok && t.stmtCtx != nil && !nr.IsTODO() {
		t.stmtCtx.flags |= unreachableStmt
	}
	// Ensure the expression is allowed to be used in t's context ([Expr.mode])
	if filtered, kind := t.IsFiltered(expr); filtered {
		_ = kind
	}
	// Record the expression node and its *Expr
	c.Info.Expressions[expr] = t
	return t
}

// IsFiltered reports whether the given node is disallowed based on e's mode.
// This is to ensure specific types of nodes don't appear in certain targets,
// such as 'when' expressions in string interpolations. A human-friendly
// name of the node is returned if filtered is false.
func (e *Expr) IsFiltered(expr ast.Expression) (filtered bool, node string) {
	return
}

func (c *Checker) checkSymbolExpr(s *ast.Symbol, allowType bool, t *Expr) {
	t.Type = InvalidType
	var (
		name = s.Identifier
		obj  = t.Context.LookupRecursive(name)
		fid  = t.Context.File
	)
	if obj == nil {
		c.fileError(klarerrs.Undefined(name, s.Range), fid)
		return
	}
	// If the target value hasn't been completed yet, typecheck it
	if Underlying(obj.typ) == nil {
		c.checkDeclaration(obj)
	}
	t.Type = obj
	switch {
	case !obj.IsTypeName():
	case t.hint != nil && t.hint.Kind() == KindFunction:
		// Allowed if t.hint is a function (making the expression an initializer)
		// 	parseInt: func(String) -> Result<Int> := Int
		//
		// Find the overload of the initializer this is referring to
	case allowType:
	case t.mode&indexLHS != 0 && obj.Kind() == KindEnum:
		// EnumType.item
		// Note that enum literals also have kind KindEnum
	default:
		// Type used as expression
		err := klarerrs.Range(klarerrs.ErrTypeAsValue, s.Range).
			SetParam("kind", kindOf(obj.typ))
		err.Label = quote(name) + " is a type, not a value"
		err.Name = name
		if obj.context != BuiltInContext {
			err.AddDetail(quote(name)+" was declared here", obj.FilePath(), obj.rang)
		}
		c.fileError(err, fid)
		t.Type = InvalidType
	}
}

func canRangeOver(k Kind) bool {
	switch k {
	case IntType, StringType, FloatType:
		return true
	default:
		return false
	}
}

func (c *Checker) checkRangeExpr(expr *ast.RangeExpression, t *Expr) {
	from, to := c.checkExprFrom(expr.From, t), c.checkExprFrom(expr.To, t)
	iterType := CommonType(from.Type, to.Type)
	reportTypeMismatch := func(a, b ast.Expression, ta, tb Type) {
		err := typeMismatch(ta, tb, b.GetRange())
		err.AddHighlight("This has type "+quoteAka(ta), a.GetRange())
		c.fileError(err, t.FileID())
		t.Type = &List{InvalidType}
	}
	if iterType == nil {
		reportTypeMismatch(expr.From, expr.To, from.Type, to.Type)
		return
	}
	// Step
	if expr.Step != nil {
		step := c.checkExprFrom(expr.Step, t)
		prevIterType := iterType
		if iterType = CommonType(iterType, step.Type); iterType == nil {
			reportTypeMismatch(expr.To, expr.Step, prevIterType, iterType)
			return
		}
	}

	// Check if we can range over the type
	kind := iterType.Kind()
	if !canRangeOver(kind) && kind != InvalidType {
		err := klarerrs.TypeError(
			klarerrs.ErrInvalidRangeType,
			expr.Range, "", from.Type.String(),
		)
		err.Label = "Can't range over this type"
		c.fileError(err, t.Context.File)
		t.Type = &List{iterType}
		return
	}

	// If we range over a string:
	// - The LHS/RHS must be a string constant of a single character
	// - '..<' isn't allowed, and
	// - There must be no step.
	switch {
	case kind != StringType:
	case expr.Step != nil:
		err := klarerrs.Range(klarerrs.ErrStepWithStringRange, expr.Operator.Range())
		err.Label = "Remove the step"
		c.fileError(err, t.Context.File)
	case expr.Operator.Kind == lexer.DotDotLessThan:
		err := klarerrs.Range(klarerrs.ErrOpenStringRange, expr.Operator.Range())
		err.Label = "Change this to '...'"
		// TODO: Hint on what end character to use instead of '..<'
		c.fileError(err, t.Context.File)
	case false:
		// TODO: Check constants for from and to
	}

	// TODO: Constant analysis for range exprs (e.g. '10...1...2')
	t.Type = &List{iterType}
}

func (c *Checker) checkGoExpr(expr *ast.GoExpression, t *Expr) {
	// The parser already checks that the expression is a call
	// TODO: Manually check the LHS of the call, then check
	// the function with the RHS. The parser allows `go Struct()`. Ensure
	// `Struct` is a function.
	if expr.Body != nil {
		t.Type = &Task{}
		return
	}
	arg := c.checkExprFrom(expr.Expression, t)
	t.Type = &Task{arg.Type}
}

func (c *Checker) checkAwaitExpr(expr *ast.AwaitExpression, t *Expr) {
	arg := c.checkExprFrom(expr, t)
	errNotTask := func(typ Type) {
		str := typ.String()
		err := klarerrs.TypeError(klarerrs.ErrTypeMismatch, expr.Range, "Task", str)
		err.Label = "This has type " + str
		c.fileError(err, t.Context.File)
	}
	switch typ := arg.Type; typ.Kind() {
	case KindTask:
		t.Type = Underlying(typ).(*Task).Result
	case KindTuple:
		// If `a: Task<A>` and `b: Task<B>`, `await (a, b)` is `(A, B)`
		tupleArg := Underlying(typ).(*Tuple)
		tupleRes := &Tuple{Items: make([]Type, len(tupleArg.Items))}
		for i, elem := range tupleArg.Items {
			// TODO:
			taskItem, ok := Underlying(elem).(*Task)
			if !ok {
				errNotTask(elem) // Not a Task
				tupleRes.Items[i] = InvalidType
				continue
			}
			tupleRes.Items[i] = taskItem.Result
		}
		t.Type = tupleRes
	case KindList:
		// If `taskList: [Task<T>]`, `await taskList` is `[T]`
		elem := Underlying(typ).(*List).Elem
		taskElem, ok := elem.(*Task)
		if !ok {
			errNotTask(elem) // Not a Task
			t.Type = &List{InvalidType}
			break
		}
		t.Type = &List{taskElem.Result}
	default:
		t.Type = InvalidType
		errNotTask(typ)
	}
}

func (c *Checker) checkIndexExpr(expr *ast.IndexExpression, t *Expr) {
	lhs := c.checkExprFrom(expr.Object, t, indexLHS)
	if lhs.Type.Kind() == InvalidType {
		t.Type = InvalidType
		return
	}
	// Types that can be indexed by dot implement [Indexer]
	indexer, ok := Underlying(lhs.Type).(Indexer)
	var err *klarerrs.Error
	if expr.Computed {
		rhs := c.checkExprFrom(expr.Property, t)
		// TODO: handle unions (union of #{Int: Andy} and [Any]
		// supports computed indexing)
		if compIndexer, ok := Underlying(lhs.Type).(ComputedIndexer); ok {
			err = compIndexer.IndexComputed(rhs.Type, t)
		} else {
			err = indexError(klarerrs.ErrInvalidComputedIndex, rhs.Type, "")
		}

		if err != nil && err.Code == klarerrs.ErrInvalidComputedIndex {
			err.Name = lhs.Type.String()
			// If the user uses a String computed index, suggest using a dot
			// index instead. (TODO: diff)
			if rhs.Type.Kind() == StringType {
				err.Code = klarerrs.ErrDotIndexRequired
				err.Label = "Type " + quote(lhs.Type.String()) +
					" must be indexed via a dot index"
			} else {
				err.Label = "Can't index type " + quote(lhs.Type.String()) +
					" using type " + quote(rhs.Type.String())
			}
		}
	} else if ok {
		// Dot-index
		field := expr.Property.(*ast.Symbol).Identifier
		err = indexer.Index(field, t)
		if o, ok := t.Type.(*Object); ok && Underlying(o.typ) == nil {
			c.checkDeclaration(o)
		}
	}

	switch {
	case !ok, t.Type == nil && err == nil:
		err := klarerrs.Node(klarerrs.ErrInvalidIndexType, expr.Object)
		err.Info = klarerrs.TypeErrorInfo{GotType: lhs.Type.String()}
		err.Label = "Can't index " + klarerrs.WithA(lhs.Type.Kind().String())
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	case err != nil:
		// Error while indexing, such as:
		// - Indexing using an unknown field
		// - Index out of range for list constants
		// - Non-constant tuple index
		if err.Code == klarerrs.ErrFieldNotFound {
			err.SetParam("type", lhs.Type.String())
		}
		err.Node = expr.Property
		err.Range = expr.Property.GetRange()
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	}
}

func (c *Checker) checkUnaryExpr(expr *ast.UnaryExpression, t *Expr) {
	rhs := c.checkExprFrom(expr.Right, t)
	t.Type = rhs.Type
	rhsKind := rhs.Type.Kind()
	switch expr.Operator.Kind {
	case lexer.Minus:
		// RHS must be an Int or Float
		if rhsKind != IntType && rhsKind != FloatType {
			got := rhs.Type.String()
			err := klarerrs.Node(klarerrs.ErrNegateNonNumeric, expr.Right)
			err.Label = "This has type " + quote(got)
			err.Info = klarerrs.TypeErrorInfo{GotType: got}
			c.fileError(err, t.Context.File)
			t.Type = InvalidType
		}
	case lexer.Not:
		// RHS must be Bool
		if rhsKind != BoolType {
			got := rhs.Type.String()
			err := klarerrs.Node(klarerrs.ErrNegateNonNumeric, expr.Right)
			err.Label = "This has type " + quote(got)
			err.Info = klarerrs.TypeErrorInfo{GotType: got}
			err.Name = expr.Operator.String()
			// Provide a hint if the user tried to negate an optional:
			// 	isNil := !optional
			if rhsKind == KindOptional {
				hintWithDiff(
					err, "To check for non-nilness, use 'expr != none'",
					&klarerrs.DeletedRange{Range: expr.Operator.Range()},
					&klarerrs.AddedString{
						Position: expr.Right.GetRange().End.Add(0, 1),
						String:   "!= none",
					},
				)
			}
			c.fileError(err, t.Context.File)
		}
		t.Type = BoolType
	default:
		panic(fmt.Sprintf("unhandled unary operator: %q", expr.Operator))
	}
}

func (c *Checker) checkBinaryExpr(expr *ast.BinaryExpression, t *Expr) {
	lhs := c.checkExprFrom(expr.Left, t)
	rhs := c.checkExpr(expr.Right, t.NewChild().withHint(lhs.Type))
	t.Type = c.checkBinaryOperation(
		expr.Operator, lhs.Type, rhs.Type,
		expr.Left, expr.Right, expr, t.FileID(),
	)
}

func (c *Checker) checkBinaryOperation(op ast.Operator, lhs, rhs Type,
	lhsNode, rhsNode ast.Expression, fullExpr *ast.BinaryExpression, fid FileID,
) (result Type) {
	lhsKind, rhsKind := lhs.Kind(), rhs.Kind()
	mismatchedOperandsError := func() *klarerrs.Error {
		err := klarerrs.Node(klarerrs.ErrOperandTypeMismatch, rhsNode)
		err.Name = op.String()
		err.AddHighlight("This has type "+quoteAka(lhs), lhsNode.GetRange())
		err.Label = "This has type " + quoteAka(rhs)
		c.fileError(err, fid)
		return err
	}
	// TODO: handle unions
	// TODO: Respect t.hint. If the hint is `Int | Float`, `Int(2) and 5.5` is allowed
	switch op.Kind {
	case lexer.AndAnd, lexer.OrOr:
		result = BoolType
		for _, side := range [...]struct {
			t    Type
			node ast.Expression
		}{{lhs, lhsNode}, {rhs, rhsNode}} {
			if Compatible(side.t, BoolType) {
				continue
			}
			err := typeMismatch(BoolType, side.t, side.node.GetRange())
			err.Code = klarerrs.ErrNonBoolLogical
			err.Name = op.String()
			err.Label = "This has type " + quote(lhs.String())
			c.fileError(err, fid)
			return InvalidType
		}
	case lexer.Plus:
		// Int, Float, String, List, Map
		if result = CommonType(lhs, rhs); result == nil {
			c.fileError(mismatchedOperandsError(), fid)
			return InvalidType
		}
		switch result.Kind() {
		case IntType, StringType, FloatType, KindList, KindMap:
		default:
			err := klarerrs.Node(klarerrs.ErrInvalidAdditionType, rhsNode)
			err.AddHighlight("", lhsNode.GetRange())
			err.Label = "These have type " + quoteAka(result)
			err.Name = result.Kind().String()
			if result.Kind() == KindTuple {
				// Hint on tuple + tuple to use (tuple1..., tuple2...) instead
				// TODO: Can we use a line diff instead? (old line - new line)
				// Do this after we're able to print AST nodes
				err.HintWithDiff(
					"To concatenate tuples, spread them into a single tuple", klarerrs.NewDiff(
						"",
						klarerrs.AddedString{Position: lhsNode.GetRange().Start, String: "("},
						klarerrs.AddedString{Position: lhsNode.GetRange().End, String: "...,"},
						klarerrs.DeletedRange{ranges.Range{op.Range().Start, op.Range().End.Add(0, 1)}},
						klarerrs.AddedString{Position: rhsNode.GetRange().End, String: "...)"},
					),
				)
			}
			c.fileError(err, fid)
		}
	case lexer.Asterisk:
		// Int, Float, String * Int
		if lhsKind == StringType {
			if !Compatible(rhs, IntType) {
				err := typeMismatch(IntType, rhs, rhsNode.GetRange())
				err.Code = klarerrs.ErrInvalidStringMult
				err.Label = "Expected an Int, but this is " + quote(rhs.String())
				err.AddHighlight(
					"This has type "+quote(lhs.String()), // String
					lhsNode.GetRange(),
				)
				c.fileError(err, fid)
				return InvalidType
			}
			return StringType
		} else if lhsKind == IntType && rhsKind == StringType {
			// Wrong order. String * Int, not Int * String
			var err *klarerrs.Error
			if fullExpr != nil {
				err = klarerrs.Node(klarerrs.ErrIntTimesString, fullExpr)
				err.Label = "Switch these operands"
			} else {
				err = klarerrs.Node(klarerrs.ErrIntTimesString, rhsNode)
				err.Label = "The operand on the right should be the Int"
			}
			c.fileError(err, fid)
			return StringType
		}
		fallthrough
	case lexer.Minus, lexer.Slash, lexer.Percent, lexer.Caret,
		lexer.LessThan, lexer.LessEqualTo, lexer.GreaterThan, lexer.GreaterEqualTo:
		// Int, Float
		if result = CommonType(lhs, rhs); result == nil {
			// Mismatched operands
			c.fileError(mismatchedOperandsError(), fid)
			return InvalidType
		}
		if kind := result.Kind(); kind != IntType && kind != FloatType {
			var err *klarerrs.Error
			if fullExpr != nil {
				err = klarerrs.Node(klarerrs.ErrInvalidArithType, fullExpr)
			} else {
				err = klarerrs.Node(klarerrs.ErrInvalidArithType, rhsNode)
				err.AddHighlight("", lhsNode.GetRange())
			}
			err.Name = op.String()
			err.Label = "These have type " + quote(result.String())
			c.fileError(err, fid)
			return InvalidType
		}
		switch op.Kind {
		case lexer.LessThan, lexer.LessEqualTo, lexer.GreaterThan, lexer.GreaterEqualTo:
			result = BoolType
		}
	case lexer.EqualEqual, lexer.NotEqual:
		compType := CommonType(lhs, rhs)
		if compType == nil {
			c.fileError(mismatchedOperandsError(), fid)
			return BoolType
		}
		// In Klar, all types can be compared for equality
		if compType.Kind() == KindFunction {
			// If comparing functions, we can report an error if different function
			// references are compared, because the result is always known.
			// 	func a() = 1
			// 	func b() = 1
			// 	_ = a == b
			// If at least 1 is a variable, we won't report an error.
			// 	fn := a
			//  _ = fn == b // Valid without constant analysis
		}
		return BoolType
	case lexer.And, lexer.Or:
		// Distributive: any type, but both sides must be the same
		if result = CommonType(lhs, rhs); result == nil {
			// Both operands must have the same type
			c.fileError(mismatchedOperandsError(), fid)
			return InvalidType
		}
		// TODO: Ensure they are used in another binary operation. With that
		// requirement, the type of this expression is trivial.
		// 	Allowed: a and b > 5
		// 	Not allowed: _ = a and b
	case lexer.In, lexer.NotIn:
		// T in [T], K in #{K: V}
		result = BoolType
		switch rhsKind {
		case KindMap:
			mp, isTyped := Underlying(rhs).(*Map)
			if !isTyped {
				// If the RHS is untyped, its value is #{}. `_ in #{}` is always false
				// TODO: error. When saying "always false", be aware to say "always true"
				// if !in is used.
				return
			}
			if !Compatible(lhs, mp.Key) {
				err := typeMismatch(mp.Key, lhs, lhsNode.GetRange())
				err.AddHighlight(
					"This map has type "+quote(mp.String()),
					rhsNode.GetRange(),
				)
				// If the LHS is a map value instead of a key, show a hint (V in #{K: V})
				if Compatible(lhs, mp.Value) {
					var not string
					if op.Kind == lexer.NotIn {
						not = "n't"
					}
					err.Hintf(
						"The %s operator checks if a key, not a value, is%s in a map",
						op, not,
					)
				}
				c.fileError(err, fid)
			}
		case KindList:
			list, isTyped := Underlying(rhs).(*List)
			if !isTyped {
				// TODO: error
				return
			}
			if !Compatible(lhs, list.Elem) {
				err := typeMismatch(list.Elem, lhs, lhsNode.GetRange())
				err.AddHighlight(
					"This list has type "+quote(list.String()),
					rhsNode.GetRange(),
				)
				c.fileError(err, fid)
			}
		default:
			err := klarerrs.Node(klarerrs.ErrInvalidInOperand, rhsNode)
			err.Label = "This has type " + quote(rhs.String())
		}
	default:
		panic(fmt.Sprintf("unhandled binary operator: %q", op))
	}
	return result
}

func (c *Checker) checkRelationalExpr(expr *ast.RelationalExpression, t *Expr) {
	for i, op := range expr.Operators {
		lhsNode, rhsNode := expr.Expressions[i], expr.Expressions[i+1]
		lhs := c.checkExprFrom(lhsNode, t)
		rhs := c.checkExpr(rhsNode, t.NewChild().withHint(lhs.Type))
		c.checkBinaryOperation(op, lhs.Type, rhs.Type, lhsNode, rhsNode, nil, t.FileID())
	}
	t.Type = BoolType
}

func (c *Checker) checkSliceExpr(expr *ast.SliceExpression, t *Expr) {
	lhs := c.checkExprFrom(expr.Object, t)
	for _, part := range [...]ast.Expression{expr.From, expr.To} {
		if part == nil {
			continue
		}
		e := c.checkExprFrom(part, t)
		if e.Kind() != IntType {
			err := klarerrs.Node(klarerrs.ErrNonNumericIndex, part)
			err.Label = "Can't slice a list using type " + quoteAka(e.Type)
			err.Info = klarerrs.TypeErrorInfo{IntType.String(), e.Type.String()}
			c.fileError(err, t.FileID())
		}
	}
	switch lhs.Type.Kind() {
	case KindList:
		t.Type = lhs.Type
	case KindTuple:
		// TODO: Check constants and slice
		t.Type = lhs.Type
	default:
		t.Type = lhs.Type
	}
}

func (c *Checker) checkStructDotInitExpr(expr *ast.StructDotInit, t *Expr) {
	switch {
	case t.hint != nil && t.hint.Kind() == KindStruct:
		t.Type = t.hint
	case t.hint != nil:
		t.Type = InvalidType
	default:
		t.Type = &UntypedInit{kind: KindStruct, Node: expr, Params: expr.Params}
	}
	// Check the parameters once its type is inferred
	c.queue(func() { c.checkStructDotInitParams(expr, t) }, false)
}

func (c *Checker) checkAssertExpr(expr *ast.AssertExpression, t *Expr) {
	lhs := c.checkExprFrom(expr.Expression, t)
	switch lhs.Kind() {
	case KindOptional:
		t.Type = Underlying(lhs.Type).(*Optional).Elem
	case KindResult:
		t.Type = Underlying(lhs.Type).(*Result).Success
	default:
		err := klarerrs.Node(klarerrs.ErrInvalidAssertType, expr.Expression)
		err.Label = "This has type " + quote(lhs.Type.String())
		c.fileError(err, t.Context.File)
		t.Type = lhs.Type
		return
	}
}

func (c *Checker) checkTryExpr(expr *ast.TryExpression, t *Expr) {
	rhs := c.checkExprFrom(expr.Expression, t)
	if rhs.Kind() == InvalidType {
		t.Type = InvalidType
		return
	}
	if rhs.Kind() != KindResult {
		err := klarerrs.Node(klarerrs.ErrNonResultInTry, expr.Expression)
		err.Label = "This has type " + quote(rhs.Type.String())
		c.fileError(err, t.Context.File)
		t.Type = rhs.Type
		return
	}
	res := Underlying(rhs.Type).(*Result)
	t.Type = res.Success
}

func (c *Checker) checkListCastExpr(expr *ast.ListCastExpression, t *Expr) {
	elem := c.parseType(expr.Type, t.Context)
	// TODO: Check params
	t.Type = &List{elem}
}

func (c *Checker) checkMapCastExpr(expr *ast.MapCastExpression, t *Expr) {
	key := c.parseType(expr.KeyType, t.Context)
	val := c.parseType(expr.ValueType, t.Context)
	// TODO: Check params
	t.Type = &Map{key, val}
}

func (c *Checker) checkLambdaExpr(expr *ast.LambdaExpression, t *Expr) {
	bodyCtx := NewContext(t.Context, t.Context.File)
	sig := &Lambda{}
	// For now, we're only collecting the explicit types for params and returns
	for _, pair := range expr.Params {
		if pair.Type == nil {
			continue
		}
		typ, variadic := c.parseTypeOrVariadic(pair.Type, t.Context)
		if variadic {
			sig.Variadic = true
			// Ensure this is the last param
		}
		// sig.Params is lazy-initialized only if any explicit types were provided
		if sig.Params == nil {
			sig.Params = make([]Type, 0, len(expr.Params))
		}
		for range max(len(pair.Keys), 1) {
			sig.Params = append(sig.Params, typ)
		}
	}
	// TODO: params and checking return type
	c.queue(func() {
		// There was an error before this queued function, such as wrong
		// param counts or types.
		if t.Type == InvalidType {
			return
		}
		// At the time this is run, the function's params and return type
		// should be resolved.
		if t.hint == nil {
			// Untyped lambda. Ensure we have all the param types, or report an error.
			// Invalid:
			// 	_ = func a, b {}
			// Valid:
			//  _ = func(a: Int, b: Int) -> Int {}
		}
		c.checkBlock(expr.Block.Body, newStmtContext(bodyCtx, t.Context.File, 0))
		sig.Complete = true
	}, true)
	t.Type = sig
}

const PipelineResultName = "value"

func (c *Checker) checkPipelineExpr(expr *ast.PipelineExpression, t *Expr) {
	var (
		valObj = NewObject(
			PipelineResultName, t.Context.File, expr.Range, c.module, nil,
		)
		valVar      = NewVariable(valObj, PipelineVar, nil)
		pipelineCtx = NewContext(t.Context, t.Context.File)
	)
	for i, step := range expr.Steps {
		if ret, ok := step.(*ast.ReturnStatement); ok {
			if (t.mode & exprStmt) != 0 {
				// A `return` in a pipeline is only allowed in expresion statements.
				// Not allowed:
				// 	_ = a() |> b |> return
				err := klarerrs.Node(klarerrs.ErrReturnInPipelineExpr, ret)
				err.Label = "This is only allowed when the pipeline is a statement"
				c.fileError(err, t.Context.File)
			}
			c.checkReturnStmt(ret, t.stmtCtx)
			continue
		}
		// TODO: Ensure each step is a call, and pass `value` as a param
		e := t.NewChild()
		e.Context = pipelineCtx
		c.checkExpr(step.(ast.Expression), e)
		valVar.Type = e.Type // Set `value` to the type of the last step
		if i == 0 {
			pipelineCtx.Declare(valObj)
		}
	}
	t.Type = valVar.Type
}

func (c *Checker) checkObjectPipeline(expr *ast.ObjectPipeline, t *Expr) {
	obj := c.checkExprFrom(expr.Object, t)
	for _, step := range expr.Steps {
		lhs := t.NewChild()
		switch step := step.(type) {
		case *ast.CallExpression:
			c.checkIndexExpr(&ast.IndexExpression{
				Object:   expr.Object,
				Property: step.Callee,
			}, lhs)
			c.checkCallArgs(lhs.Type, step, t.NewChild())
		case *ast.AssignmentStatement:
			c.checkIndexExpr(&ast.IndexExpression{
				Object:   expr.Object,
				Property: step.Assignee[0],
			}, lhs)
			rhs := t.NewChild().withHint(lhs.Type)
			c.checkExpr(step.Values[0], rhs)
			c.checkAssignment(
				lhs.Type, rhs.Type, step.Assignee[0], step.Values[0],
				step.Operator.Uncompound(), t.FileID(),
			)
		}
	}
	_ = obj
}

// canSpreadTuple checks if the provided tuple can be spread into a list.
// That is true if the types of all of the tuple's items are common.
// If they are in common, the common type is returned. Otherwise,
// canSpreadTuple returns nil and an error if the tuple can't be spread.
func canSpreadTuple(t *Tuple) (commonType Type, err *klarerrs.Error) {
	if len(t.Items) == 0 {
		return nil, &klarerrs.Error{
			Code:  klarerrs.ErrSpreadEmptyTuple,
			Label: "This tuple is empty",
		}
	}
	for _, item := range t.Items {
		if commonType = CommonType(commonType, item); commonType == nil {
			return nil, &klarerrs.Error{
				Code:  klarerrs.ErrRestUncommonTuple,
				Label: "This tuple has type " + t.String(),
			}
		}
	}
	return commonType, nil
}
