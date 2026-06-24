package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type stmtContext struct {
	ctx        *Context
	returns    *[]returnStmt
	loopLabels map[string]*loopLabel
	flags      stmtFlags
	collector  *stmtCollector
}

type returnStmt struct {
	expr *Expr
	pos  ranges.Range
}

type loopLabel struct {
	pos  ranges.Range
	used bool
}

type stmtFlags uint8

const (
	allowReturn   stmtFlags = 1 << iota // Function body
	allowNextStop                       // Allow 'next' and 'stop' (for/while/when)
	finalWhenCase
	unreachable
	braceless // Body of a braceless 'when' case
	allowForwardDecl
)

func newStmtContext(ctx *Context, fid FileID, flags stmtFlags) *stmtContext {
	return &stmtContext{
		ctx:        ctx,
		flags:      flags,
		returns:    new([]returnStmt),
		loopLabels: make(map[string]*loopLabel),
		collector:  &stmtCollector{ctx: ctx, fid: fid},
	}
}

func newChildStmtContext(parentSctx *stmtContext,
	childCtx *Context, flags stmtFlags,
) *stmtContext {
	return &stmtContext{
		ctx:        childCtx,
		returns:    parentSctx.returns,
		loopLabels: parentSctx.loopLabels,
		flags:      parentSctx.flags | flags,
		collector:  &stmtCollector{ctx: childCtx, fid: childCtx.File},
	}
}

// Reports an error if the label already exists
//
// Redeclared labels in the same function (not just block/context) defeat the entire
// purpose of labels:
//
//	for i in 1...10 :loop {
//		for j in 'a'...'z' :loop {
//			stop :loop // Which loop?
//		}
//	}
func (sctx *stmtContext) declareLabel(name string, r ranges.Range) (err *klarerrs.Error) {
	if other, ok := sctx.loopLabels[name]; ok {
		err := klarerrs.Range(klarerrs.ErrRedeclaredLabel, r)
		err.Label = "A label named " + quote(name) + " was already defined"
		// If this is the top-level context, don't call it a function
		if sctx.ctx.index > 0 {
			err.SetParam("isFunc", true)
			err.Label += " in this function"
		}
		err.AddDetail("It was already defined here", "", other.pos)
		return err
	}
	sctx.loopLabels[name] = &loopLabel{pos: r}
	return nil
}

func (c *Checker) checkBlock(stmts []ast.Statement, sctx *stmtContext) {
	defer func(oldFlags stmtFlags) { sctx.flags = oldFlags }(sctx.flags)
	sctx.flags |= allowForwardDecl

	// Declare functions and types first
	var normalStmts []ast.Statement
	for _, stmt := range stmts {
		if canForwardDeclareInFunc(stmt) {
			c.checkStmt(stmt, sctx) // Declare without checking them
		} else {
			normalStmts = append(normalStmts, stmt)
		}
	}
	if len(normalStmts) < len(stmts) {
		// Actually check the declarations. Similar to [Checker.Check]
		c.checkDirectCycles(sctx.ctx)
		c.checkContextDecls(sctx.ctx, sctx.collector.methods, sctx.collector.inits)
	}

	// Check everything else in the block, including variable declarations.
	for _, stmt := range normalStmts {
		c.checkStmt(stmt, sctx)
	}
}

func canForwardDeclareInFunc(stmt ast.Statement) bool {
	switch stmt.(type) {
	case *ast.FunctionDeclaration, *ast.FuncAliasDeclaration, ast.TypeDeclaration:
		return true
	default:
		return false
	}
}

func (c *Checker) checkStmt(stmt ast.Statement, sctx *stmtContext) {
	defer c.runDelayed(len(c.delayed))

	fid := sctx.ctx.File
	switch stmt := stmt.(type) {
	case *ast.ExpressionStatement:
		expr := c.checkExpr(stmt.Expression, newExprFromStmtCtx(sctx, 0))
		switch {
		case (sctx.flags & braceless) != 0:
		// TODO: find a way to return the value type
		case !isAllowedAsStmt(stmt.Expression):
			// Unused expression value
			c.fileError(klarerrs.Node(klarerrs.ErrUnusedValue, stmt), fid)
		// TODO: exclude InvalidType from these errors?
		case c.Options.UseAllValues && expr.Type.Kind() != NothingType:
		// Expression returns something and isn't used
		case c.Options.CheckAllResults && expr.Type.Kind() == KindResult:
		// Unchecked result
		default:
			return
		}
	case *ast.BadExpression:
		return

	case ast.TypeDeclaration:
		c.declareType(stmt, sctx.collector, false, nil)
	case *ast.FunctionDeclaration:
		c.declareFunc(stmt, sctx.collector, false, nil)
	case *ast.FuncAliasDeclaration:
		c.declareFuncAlias(stmt, sctx.collector, false, nil)
	case *ast.VariableDeclaration:
		c.declareVars(stmt, sctx.collector, false, nil)
	case *ast.AssignmentStatement:

	case ast.ModifierDeclaration:
		// TODO: Could a main.klar file reach a public statement at the top-level?
		panic("invalid AST: public declaration must be at top-level")
	case *ast.ForStatement:
		c.checkForStmt(stmt, sctx)
	case *ast.WhileStatement:
		c.checkWhileStmt(stmt, sctx)
	case *ast.StopStatement:
		c.checkControlStmt(stmt, stmt.Label, sctx)
	case *ast.NextStatement:
		c.checkControlStmt(stmt, stmt.Label, sctx)
	case *ast.ReturnStatement:
		expr := c.checkExpr(stmt.Value, newExprFromStmtCtx(sctx, 0))
		*sctx.returns = append(*sctx.returns, returnStmt{
			expr: expr, pos: stmt.Value.GetRange(),
		})
	default:
		panic(fmt.Sprintf("unhandled statement node: %T", stmt))
	}
	// If we're checking a single statement, forward declarations aren't
	// allowed, so we need to typecheck declarations immediately.
	if canForwardDeclareInFunc(stmt) && (sctx.flags&allowForwardDecl) == 0 {
		c.checkDirectCycles(sctx.ctx) // Only self-cycles are reachable here
		c.checkContextDecls(sctx.ctx, sctx.collector.methods, sctx.collector.inits)
	}
}

func (c *Checker) checkWhileStmt(stmt *ast.WhileStatement, sctx *stmtContext) {
	if stmt.Condition != nil {
		cond := c.checkExpr(stmt.Condition, newExprFromStmtCtx(sctx, 0))
		if cond.Type.Kind() != BoolType {
			gotType := cond.Type.String()
			err := klarerrs.TypeError(
				klarerrs.ErrNonBoolWhileCond, stmt.Condition.GetRange(),
				BoolType.String(), gotType,
			)
			err.Label = "This has type " + quote(gotType)
			c.fileError(err, sctx.ctx.File)
		}
	}
	// Optional loop label
	if lb := stmt.Label; lb != nil {
		if err := sctx.declareLabel(lb.Name, lb.GetRange()); err != nil {
			c.fileError(err, sctx.ctx.File)
		}
	}
}

func (c *Checker) checkControlStmt(stmt ast.Statement,
	label *ast.Identifier, sctx *stmtContext,
) {
	fid := sctx.ctx.File
	if (sctx.flags & allowNextStop) == 0 {
		c.fileError(klarerrs.Node(klarerrs.ErrMisplacedControlStmt, stmt), fid)
		return
	}
	if label != nil {
		labelDef, ok := sctx.loopLabels[label.Name]
		if !ok {
			err := klarerrs.Node(klarerrs.ErrLabelUndefined, label)
			err.Label = "Label :" + label.Name + " doesn't exist"
			if sctx.ctx.index > 0 {
				err.SetParam("isFunc", true)
				err.Label += " in this function"
			}
			c.fileError(err, fid)
			return
			// TODO: More specific error if the label is in an outside function
		}
		labelDef.used = true
	}
}

const MaxLoopVars = 2

// TODO: Factor out this function so it can be used for 'for' expressions.
func (c *Checker) checkForStmt(stmt *ast.ForStatement, sctx *stmtContext) {
	fid := sctx.ctx.File
	// For now, we don't actually care how many there actually are. We just need
	// to know whether there are 2 vs 1. We will report errors when there are more
	// than 2 when we declare the vars.
	var numVars int // Can be 0
	if numPairs := len(stmt.Variables); numPairs > 1 {
		numVars = numPairs
	} else if numPairs > 0 {
		numVars = len(stmt.Variables[0].Keys) + (numPairs - 1)
	}
	numVars = min(numVars, MaxLoopVars) // Will always be in range [0, MaxLoopVars]

	iterExpr := c.checkExpr(stmt.Expression, newExprFromStmtCtx(sctx, 0))
	varTypes, err := c.isIterable(iterExpr.Type, numVars)
	if err != nil {
		err.Range = stmt.Expression.GetRange()
		c.fileError(err, fid)
		// The loop variables will still be declared with types [InvalidType]
	}
	if iterExpr.Type.Kind() == IntType && numVars > 1 {
		err := klarerrs.Slice(klarerrs.ErrMultipleIntIterVars, stmt.Variables)
		err.AddHighlight("The iterator has type Int", stmt.Expression.GetRange())
		err.Label = "Multiple loop variables aren't allowed"
		c.fileError(err, fid)
	}
	var i int
	for _, pair := range stmt.Variables {
		var typ Type = InvalidType
		// Use the user-provided type annotation, if any.
		//
		// TODO: Should we keep allowing users to declare explicit types for
		// loop variables? They are completely known without them, and an error
		// is raised if the annotation is incompatible. Annotations will only
		// be useful for downcasting `for i: Animal in [Cat](...)`
		if pair.Type != nil {
			typ = c.parseType(pair.Type, sctx.ctx)
			// TODO: Check that the annotation is compatible with the actual loop type
		}
		for _, key := range pair.Keys {
			if i >= MaxLoopVars {
				// Currently in the language, there will never be more than
				// 2 loop variables. Unless we add custom iterators to the language,
				// however it's unlikely because lists are enough.
				c.fileError(klarerrs.Node(klarerrs.ErrOver2LoopVars, key), fid)
				break
			}
			if typ == InvalidType {
				typ = varTypes[i] // Default type from the loop expression
			}
			// TODO: destructure and declare
			i++
		}
	}
	// Optional loop label
	if lb := stmt.Label; lb != nil {
		if err := sctx.declareLabel(lb.Name, lb.GetRange()); err != nil {
			c.fileError(err, sctx.ctx.File)
		}
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

func (c *Checker) checkAssignment(e *Expr) {
}
