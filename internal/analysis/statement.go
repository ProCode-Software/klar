package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type Return struct {
	Node       ast.ReturnStatement
	ReturnType Type
}
type contextInfo struct {
	IsLoop         bool
	ReturnType     Type
	InferredReturn bool
	doesReturn     bool
}

func (c *Checker) CheckStatements(
	body []ast.Statement, ctxInfo contextInfo, ctx context,
) (returns []Return) {
	ctxInfo.doesReturn = !ctxInfo.InferredReturn && ctxInfo.ReturnType == nil
	for i, stmt := range body {
		switch stmt := stmt.(type) {
		case ast.VariableDeclaration:
			c.CheckVarDecl(stmt, ctx)
		case ast.AssignmentStatement:
		case ast.ForStatement:
		case ast.NextStatement:
		case ast.ReturnStatement:
		case ast.UpdateStatement:
		case ast.ExpressionStatement:
			switch expr := stmt.Expression.(type) {
			// Allowed expressions as statements
			case ast.WhenExpression, ast.CallExpression, ast.BadExpression:
			default:
				// Unused statement
				err := errors.NewTypeErr(errors.ErrUnusedValue, expr.Base().Range, nil)
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

func (c *Checker) CheckVarDecl(decl ast.VariableDeclaration, ctx context) {
	var explType, actualType Type
	name := decl.Identifier
	if decl.ExplicitType != nil {
		explType = c.ParseType(decl.ExplicitType, ctx)
		_, ok := c.CheckCompatible(explType, decl.Value, ctx)
		if !ok {
			c.TypeMismatch(errors.ErrWrongAssignType, name, actualType, explType)
			actualType = explType
		}
	} else {
		// Infer type
		actualType = c.InferType(decl.Value, ctx)
	}
	if ok := ctx.Declare(name, decl.Constant, actualType, decl.Base().Range); !ok {
		c.ErrRedeclared(errors.ErrRedeclaredVar, name, decl.Base().Range, "variable", ctx)
	}
}
