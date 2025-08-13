package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ast/typed"
	"github.com/ProCode-Software/klar/internal/errors"
)

type Return struct {
	Node       *ast.ReturnStatement
	ReturnType Type
}
type contextInfo struct {
	IsLoop         bool
	ReturnType     Type
	InferredReturn bool
	doesReturn     bool
}

func (c *Checker) CheckStatements(body []ast.Statement, ctx context) (
	stmts []typed.Statement, returns []Return,
) {
	var unreachableStmt string
	for i, stmt := range body {
		if unreachableStmt != "" {
			// Statement after return
			err := errors.ParseError{
				ErrorCode: errors.ErrProvenUnreachable,
				Range:     stmt.GetRange(),
				Params:    errors.ErrorParams{"type": unreachableStmt},
			}
			// Hint if user has line break between return and expression
			if unreachableStmt == "return" &&
				body[i-1].(*ast.ReturnStatement).Value == nil {
				if _, ok := stmt.(*ast.ExpressionStatement); ok {
					err.Hint("Line breaks aren't allowed between return statements; remove the newline if you meant to return this expression.")
				}
			}
			c.Error(err)
			break
		}
		switch stmt := stmt.(type) {
		case *ast.VariableDeclaration:
			c.CheckVarDecl(stmt, ctx)
		case *ast.AssignmentStatement:
		case *ast.ForStatement:
		case *ast.NextStatement:
			unreachableStmt = "next"
		case *ast.ReturnStatement:
			unreachableStmt = "return"
		case *ast.BreakStatement:
			unreachableStmt = "break"
		case *ast.UpdateStatement:
		case *ast.ExpressionStatement:
			switch expr := stmt.Expression.(type) {
			// Allowed expressions as statements
			case *ast.WhenExpression, *ast.CallExpression, *ast.BadExpression:
			default:
				// Unused statement
				err := errors.Range(errors.ErrUnusedValue, expr.GetRange())
				if !ctx.IsRoot() && i == len(body)-1 {
					// TODO: only show if it's a valid type
					err.Hint("Did you mean to return this expression?")
				}
				c.Error(err)
				c.InferType(expr, ctx) // Just check it
				continue
			}
		}
	}
	return
}

func (c *Checker) CheckVarDecl(decl *ast.VariableDeclaration, ctx context) {
	/* var explType, actualType Type
	name := decl.Variables.(*ast.Symbol).Identifier // todo: implement other types
	if decl.ExplicitType != nil {
		explType = c.ParseType(decl.ExplicitType, ctx)
		_, ok := c.CheckCompatibleExpr(explType, decl.Value, ctx)
		if !ok {
			c.TypeMismatch(errors.ErrWrongAssignType, name, actualType, explType)
			actualType = explType
		}
	} else {
		// Infer type
		actualType = c.InferType(decl.Value, ctx)
	}
	if ok := ctx.Declare(name, decl.Constant, actualType, decl.GetRange()); !ok {
		c.ErrRedeclared(errors.ErrRedeclaredVar, name, decl.GetRange(), "variable", ctx)
	} */
}
