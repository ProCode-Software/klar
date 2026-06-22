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
	case *ast.UnaryExpression:
	case *ast.NilLiteral:
	// TODO: Use [ConstValue]s
	case *ast.StringLiteral:
		t.Type = StringType
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
	case *ast.CallExpression:
	case *ast.EnumLiteral:
	case *ast.WhenExpression:
	case *ast.TupleType:
	case *ast.LambdaExpression:
	case *ast.RangeExpression:
	case *ast.RestExpression:
	// TODO: Check for rest expressions where they are allowed, so we can report an error if they are reached here.
	// They are allowed in:
	// - Calls
	// - Tuples
	// - Lists
	// - Maps
	case *ast.PipelineExpression:
	case *ast.BadExpression:
		t.Type = InvalidType
	case *ast.SliceExpression:
	case *ast.ParenExpression:
		c.checkExpr(expr.Expression, t)
	case *ast.RegexLiteral:
		t.Type = RegExType
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
		err.AddDetail(quote(name)+" was declared here", obj.FilePath(), obj.rang)
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

func (c *Checker) checkRangeExpr(expr *ast.RangeExpression, e *Expr) {
	from := c.checkExpr(expr.From, newChildExpr(e, 0))
	kind := from.Type.Kind()
	// Check if we can range over the type
	if !canRangeOver(kind) && kind != InvalidType {
		err := klarerrs.TypeError(klarerrs.ErrInvalidRangeType, expr.Range, "", from.Type.String())
		err.Label = "Can't range over this type"
		c.fileError(err, e.Context.File)
		e.Type = &List{from.Type}
		return
	}
	to := c.checkExpr(expr.To, newChildExprWithHint(e, from.Type, 0))
	step := c.checkExpr(expr.Step, newChildExprWithHint(e, from.Type, 0))
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
		c.fileError(err, e.Context.File)
	case expr.Operator.Kind == lexer.DotDotLessThan:
		err := klarerrs.Range(klarerrs.ErrOpenStringRange, expr.Operator.Range())
		err.Label = "Change this to '...'"
		// TODO: Hint on what end character to use instead of '..<'
		c.fileError(err, e.Context.File)
	case false:
		// TODO: Check constants for from and to
	}

	// TODO: Constant analysis for range exprs (e.g. '10...1...2')
	e.Type = &List{from.Type}
}

func (c *Checker) checkGoExpr(expr *ast.GoExpression, e *Expr) {
	// The parser already checks that the expression is a call
	// TODO: Manually check the LHS of the call, then check
	// the function with the RHS. The parser allows `go Struct()`. Ensure
	// `Struct` is a function.
	arg := c.checkExpr(expr, newChildExpr(e, 0))
	e.Type = &Task{arg.Type}
}

func (c *Checker) checkAwaitExpr(expr *ast.AwaitExpression, e *Expr) {
	arg := c.checkExpr(expr, newChildExpr(e, 0))
	errNotTask := func(t Type) {
		str := t.String()
		err := klarerrs.TypeError(klarerrs.ErrTypeMismatch, expr.Range, "Task", str)
		err.Label = "This has type " + str
		c.fileError(err, e.Context.File)
	}
	switch t := arg.Type; t.Kind() {
	case KindTask:
		e.Type = Underlying(t).(*Task).Result
	case KindTuple:
		// If `a: Task<A>` and `b: Task<B>`, `await (a, b)` is `(A, B)`
		tupleArg := Underlying(t).(Tuple)
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
		e.Type = tupleRes
	case KindList:
		// If `taskList: [Task<T>]`, `await taskList` is `[T]`
		elem := Underlying(t).(*List).Elem
		taskElem, ok := elem.(*Task)
		if !ok {
			errNotTask(elem) // Not a Task
			e.Type = &List{InvalidType}
			break
		}
		e.Type = &List{taskElem.Result}
	default:
		e.Type = InvalidType
		errNotTask(t)
	}
}

func (c *Checker) isIterable(t Type, numVars int) (varTypes []Type, err *klarerrs.Error) {
	if numVars > 2 {
		panic(fmt.Sprintf("isIterable(_, numVars): expected numVars <= 2, got %d", numVars))
	}
	if numVars == 0 {
		// Still check if the type is iterable
		switch t.Kind() {
		case KindList, KindMap, StringType, IntType:
			return []Type{}, nil
		}
		// Fallthrough
	}
	switch t.Kind() {
	case KindList:
		t := Underlying(t).(*List)
		if numVars == 2 {
			return []Type{IntType, t.Elem}, nil
		}
		return []Type{t.Elem}, nil
	case KindMap:
		t := Underlying(t).(*Map)
		if numVars == 2 {
			return []Type{t.Key, t.Value}, nil
		}
		return []Type{t.Key}, nil
	case StringType:
		if numVars == 2 {
			return []Type{IntType, StringType}, nil
		}
		return []Type{StringType}, nil
	case IntType:
		if numVars == 2 {
			return []Type{IntType, InvalidType}, nil // Up to 1 loop variable is allowed
		}
		return []Type{IntType}, nil
	// TODO: Allow unions
	// If `a: String | [Any]` and `for i, v in a`, `(i, v)` is `(Int, String | Any)`

	// Not iterable, but if their underlying types are iterable, show a hint about unwrapping
	case KindResult:
		success := Underlying(t).(*Result).Success
		if varTypes, err = c.isIterable(success, numVars); err != nil {
			break // Underlying type isn't iterable
		}
		err = klarerrs.TypeError(klarerrs.ErrUnwrapRequired, ranges.Range{}, "", t.String())
		err.SetParam("kind", "Result")
		return varTypes, err
	case KindOptional:
		concrete := Underlying(t).(*Optional).Elem
		if varTypes, err = c.isIterable(concrete, numVars); err != nil {
			break // Underlying type isn't iterable
		}
		err = klarerrs.TypeError(klarerrs.ErrUnwrapRequired, ranges.Range{}, "", t.String())
		err.SetParam("kind", "Optional")
		return varTypes, err
	case InvalidType:
		return repeatWithItem(Type(InvalidType), numVars), nil // Don't show an error
	}
	// Not iterable
	err = klarerrs.TypeError(klarerrs.ErrNotIterable, ranges.Range{}, "", t.String())
	return repeatWithItem(Type(InvalidType), numVars), err
}

// isAllowedAsStmt returns whether the given expression can be used as a statement.
func isAllowedAsStmt(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.WhenExpression, *ast.CallExpression, *ast.PipelineExpression,
		*ast.ObjectPipeline, *ast.GoExpression, *ast.AwaitExpression:
		return true
	case *ast.BadExpression:
		panic("typechecking invalid AST")
	default:
		return false
	}
}
