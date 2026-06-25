package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
)

type Expr struct {
	Type    Type
	Context *Context
	Flags   Flag
	Kind    ExprKind
	hint    Type
	stmtCtx *stmtContext
}

func NewExpr(ctx *Context, flags Flag) *Expr {
	return &Expr{Context: ctx, Flags: flags}
}

func newChildExpr(parent *Expr, flags Flag) *Expr {
	return &Expr{
		Context: parent.Context,
		Flags:   parent.Flags | flags,
		stmtCtx: parent.stmtCtx,
	}
}

func newChildExprWithHint(parent *Expr, hint Type, flags Flag) *Expr {
	return &Expr{
		Context: parent.Context,
		Flags:   parent.Flags | flags,
		stmtCtx: parent.stmtCtx,
		hint:    hint,
	}
}

func NewExprWithHint(ctx *Context, hint Type, flags Flag) *Expr {
	return &Expr{Context: ctx, hint: hint, Flags: flags}
}

func newExprFromStmtCtx(sctx *stmtContext, flags Flag) *Expr {
	return &Expr{Context: sctx.ctx, Flags: flags, stmtCtx: sctx}
}

type ExprKind int

const (
	_ ExprKind = iota
)

func (e *Expr) ConstValue() ConstValue {
	return UnknownConst{}
}

func (c *Checker) checkExpr(expr ast.Expression, t *Expr) *Expr {
	switch expr := expr.(type) {
	case *ast.BinaryExpression:
		c.checkBinaryExpr(expr, t)
	case *ast.RelationalExpression:

	case *ast.UnaryExpression:
		c.checkUnaryExpr(expr, t)
	case *ast.NilLiteral:
	// TODO: Use [ConstValue]s
	case *ast.StringLiteral:
		t.Type = StringType
		// TODO: Check interpolations
	case *ast.IntegerLiteral:
	case *ast.FloatLiteral:
		t.Type = FloatType
	case *ast.BooleanLiteral:
		t.Type = BoolType
	case *ast.Symbol:
		c.checkSymbolExpr(expr, t)
	case *ast.MapLiteral:
	case *ast.TupleLiteral:
	case *ast.ListLiteral:
	case *ast.IndexExpression:
		c.checkIndexExpr(expr, t)
	case *ast.CallExpression:
		c.checkCallExpr(expr, t)
	case *ast.EnumLiteral:
	case *ast.WhenExpression:
	case *ast.LambdaExpression:
	case *ast.RangeExpression:
		c.checkRangeExpr(expr, t)
	case *ast.RestExpression:
	// TODO: Check for rest expressions where they are allowed, so we can
	// report an error if they are reached here.
	// They are allowed in:
	// - Calls
	// - Tuples
	// - Lists
	// - Maps
	// - List slices
	case *ast.PipelineExpression:
	case *ast.BadExpression:
		t.Type = InvalidType
	case *ast.SliceExpression:
	case *ast.ParenExpression:
		c.checkExpr(expr.Expression, t)
	case *ast.RegexLiteral:
		t.Type = RegExType
		// TODO: Check interpolations
	case *ast.VersionLiteral:
		// These are only parsed within attributes.
		// TODO: Find a way to read these when applying attributes
	case *ast.ListCastExpression:
	case *ast.MapCastExpression:
	case *ast.ObjectPipeline:
	case *ast.ForExpression:
	case *ast.StructDotInit:
	case *ast.GoExpression:
		c.checkGoExpr(expr, t)
	case *ast.AwaitExpression:
		c.checkAwaitExpr(expr, t)
	case *ast.AssertExpression:
	case *ast.TryExpression:
	default:
		panic(fmt.Sprintf("unhandled expression node type: %T", expr))
	}
	if t.Type != nil {
		return t
	}
	t.Type = InvalidType
	return t
}

func (c *Checker) checkSymbolExpr(s *ast.Symbol, t *Expr) {
	t.Type = InvalidType
	var (
		name = s.Identifier
		obj  = t.Context.LookupRecursive(name)
		fid  = t.Context.File
	)
	if obj == nil {
		c.fileError(klarerrs.Undefined(name, s.Range), fid)
		return
	} else if obj.IsTypeName() {
		// Only allowed if t.hint is a function (making the expression an initializer)
		// 	parseInt: func(String) -> Result<Int> := Int
		if t.hint != nil && t.hint.Kind() == KindFunction {
			// Find the overload of the initializer this is referring to
		}
		err := klarerrs.Range(klarerrs.ErrTypeAsValue, s.Range).
			SetParam("kind", kindOf(obj.typ))
		err.Label = quote(name) + " is a type, not a value"
		err.Name = name
		if obj.context != BuiltInContext {
			err.AddDetail(quote(name)+" was declared here", obj.FilePath(), obj.rang)
		}
		c.fileError(err, fid)
		return
	}
	// If the target value hasn't been completed yet, typecheck it
	if Underlying(obj.typ) == nil {
		c.checkDeclaration(obj)
	}
	t.Type = obj.typ
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
	from := c.checkExpr(expr.From, newChildExpr(t, 0))
	kind := from.Type.Kind()
	// Check if we can range over the type
	if !canRangeOver(kind) && kind != InvalidType {
		err := klarerrs.TypeError(klarerrs.ErrInvalidRangeType, expr.Range, "", from.Type.String())
		err.Label = "Can't range over this type"
		c.fileError(err, t.Context.File)
		t.Type = &List{from.Type}
		return
	}
	// TODO: unify (`3...5.0` should have type Float, not Int)
	to := c.checkExpr(expr.To, newChildExprWithHint(t, from.Type, 0))
	step := c.checkExpr(expr.Step, newChildExprWithHint(t, from.Type, 0))
	_, _ = to, step

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
	t.Type = &List{from.Type}
}

func (c *Checker) checkGoExpr(expr *ast.GoExpression, t *Expr) {
	// The parser already checks that the expression is a call
	// TODO: Manually check the LHS of the call, then check
	// the function with the RHS. The parser allows `go Struct()`. Ensure
	// `Struct` is a function.
	arg := c.checkExpr(expr, newChildExpr(t, 0))
	t.Type = &Task{arg.Type}
}

func (c *Checker) checkAwaitExpr(expr *ast.AwaitExpression, t *Expr) {
	arg := c.checkExpr(expr, newChildExpr(t, 0))
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
		tupleArg := Underlying(typ).(Tuple)
		tupleRes := make(Tuple, len(tupleArg))
		for i, elem := range tupleArg {
			// TODO:
			taskItem, ok := Underlying(elem).(*Task)
			if !ok {
				errNotTask(elem) // Not a Task
				tupleRes[i] = InvalidType
				continue
			}
			tupleRes[i] = taskItem.Result
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
	lhs := c.checkExpr(expr.Object, newChildExpr(t, 0))
	if lhs.Type.Kind() == InvalidType {
		t.Type = InvalidType
		return
	}
	// Types that can be indexed implement [Indexer]
	indexer, ok := Underlying(lhs.Type).(Indexer)
	cantIndexErr := func() {
		err := klarerrs.Node(klarerrs.ErrInvalidIndexType, expr.Object)
		err.Info = klarerrs.TypeErrorInfo{GotType: lhs.Type.String()}
		err.Label = "Can't index " + klarerrs.WithA(lhs.Type.Kind().String())
		// err.Label = "Type " + klarerrs.Quote(lhs.Type.String()) + " can't be indexed"
		c.fileError(err, t.Context.File)
		t.Type = InvalidType
	}
	if !ok {
		// Can't index the LHS type
		cantIndexErr()
		return
	}

	var err *klarerrs.Error
	if expr.Computed {
		rhs := c.checkExpr(expr.Property, newChildExpr(t, 0))
		// TODO: handle unions (union of #{Int: Andy} and [Any]
		// supports computed indexing)
		t.Type, err = indexer.Index(rhs.Type)

		// Add LHS type information to error. And if the user uses a String
		// computed index, suggest using a dot index instead. (TODO: diff)
		if err != nil && err.Code == klarerrs.ErrInvalidComputedIndex {
			err.Name = lhs.Type.String()
			if rhs.Type.Kind() == StringType {
				err.Code = klarerrs.ErrDotIndexRequired
				err.Label = "Type " + quote(lhs.Type.String()) + " must be indexed via a dot index"
			} else {
				err.Label = "Can't index type " + quote(lhs.Type.String()) +
					" using type " + quote(rhs.Type.String())
			}
		}
	} else {
		// Dot-index
		field := expr.Property.(*ast.Symbol).Identifier
		t.Type, err = indexer.IndexDot(field)
	}
	if t.Type == nil && err == nil {
		// Still can't index type
		cantIndexErr()
		return
	}
	// Error while indexing, such as:
	// - Using a computed index for a field
	//   - TODO: Handle that here by calling IndexDot with the string if Index fails
	// - Type doesn't support computed indexing (e.g. struct)
	// - Indexing using an unknown field
	// - Index out of range for list constants
	// - Non-constant tuple index
	if err != nil {
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
	rhs := c.checkExpr(expr.Right, newChildExpr(t, 0))
	t.Type = rhs.Type
	rhsKind := rhs.Type.Kind()
	switch expr.Operator.Kind {
	case lexer.Minus:
		// RHS must be an Int or Float
		if rhsKind != IntType && rhsKind != FloatType {
			got := rhs.Type.String()
			err := klarerrs.Node(klarerrs.ErrNegateNonNumeric, expr)
			err.Label = "This has type " + quote(got)
			err.Info = klarerrs.TypeErrorInfo{GotType: got}
			c.fileError(err, t.Context.File)
			t.Type = InvalidType
		}
	case lexer.Not:
	// RHS must be Bool or optional
	default:
		panic(fmt.Sprintf("unhandled unary operator: %q", expr.Operator))
	}
}

func (c *Checker) checkBinaryExpr(expr *ast.BinaryExpression, t *Expr) {
	lhs := c.checkExpr(expr.Left, newChildExpr(t, 0))
	lhsKind := lhs.Type.Kind()
	// TODO: handle unions
	switch expr.Operator.Kind {
	case lexer.AndAnd, lexer.OrOr:
		// Bool
		if !Compatible(lhs.Type, BoolType) {
			err := typeMismatch(BoolType, lhs.Type, expr.Left.GetRange())
			err.Code = klarerrs.ErrNonBoolLogical
			err.Name = expr.Operator.String()
			err.Label = "This has type " + quote(lhs.Type.String())
			c.fileError(err, t.Context.File)
			t.Type = InvalidType
			return
		}
		_ = c.checkExpr(expr.Right, newChildExprWithHint(t, BoolType, 0))
		t.Type = BoolType
	case lexer.Plus:
	// Int, Float, String, List, Map
	case lexer.Asterisk:
		// Int, Float, String * Int
		rhs := c.checkExpr(expr.Right, newChildExpr(t, 0))
		rhsKind := rhs.Type.Kind()
		if lhsKind == StringType {
			if !Compatible(rhs.Type, IntType) {
				err := typeMismatch(IntType, rhs.Type, expr.Right.GetRange())
				err.Code = klarerrs.ErrInvalidStringMult
				err.Label = "Expected an Int, but this is " + quote(rhs.Type.String())
				err.AddHighlight(
					"This has type "+quote(lhs.Type.String()), // String
					expr.Left.GetRange(),
				)
				c.fileError(err, t.Context.File)
				t.Type = InvalidType
				return
			}
			t.Type = StringType
			break
		} else if lhsKind == IntType && rhsKind == StringType {
			// Wrong order. String * Int, not Int * String
			err := klarerrs.Node(klarerrs.ErrIntTimesString, expr)
			err.Label = "Switch these operands"
			c.fileError(err, t.Context.File)
			t.Type = StringType
			break
		}
		fallthrough
	case lexer.Minus, lexer.Slash, lexer.Percent, lexer.Caret:
		// Int, Float
		rhs := c.checkExpr(expr.Right, newChildExprWithHint(t, lhs.Type, 0))
		t.Type = rhs.Type
		if rhsKind := rhs.Type.Kind(); rhsKind != IntType && rhsKind != FloatType {
			err := klarerrs.Node(klarerrs.ErrInvalidArithType, expr)
			err.Name = expr.Operator.String()
			err.Label = "These have type " + quote(t.Type.String())
			c.fileError(err, t.Context.File)
			t.Type = InvalidType
		}
	case lexer.And, lexer.Or:
		// Distributive: any type, but both sides must be the same
		rhs := c.checkExpr(expr.Right, newChildExprWithHint(t, lhs.Type, 0))
		t.Type = rhs.Type
	case lexer.In, lexer.NotIn:
		// T in [T], K in #{K: V}
	default:
		panic(fmt.Sprintf("unhandled binary operator: %q", expr.Operator))
	}
}

func (c *Checker) checkCallExpr(expr *ast.CallExpression, t *Expr) {
	lhs := c.checkExpr(expr.Callee, newChildExpr(t, 0))
	t.Type = lhs.Type
	_ = lhs
}
