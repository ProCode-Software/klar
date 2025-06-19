package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type Return struct {
	Node       ast.ReturnStatement
	ReturnType Type
}

func (c *Checker) CheckStatements(body []ast.Statement, ctx context) (returns []Return) {
	for i, stmt := range body {
		switch stmt := stmt.(type) {
		case ast.VariableDeclaration:
			c.CheckVarDecl(stmt, ctx)
		case ast.ExpressionStatement:
			switch expr := stmt.Expression.(type) {
			case ast.WhenExpression, ast.CallExpression:
			default:
				// Unused statement
				err := errors.NewTypeErr(errors.ErrUnusedValue, expr.Base().Range, nil)
				if !ctx.IsRoot() && i == len(body)-1 {
					// TODO: only show if it's a valid type
					err.Hint("Did you mean to return this expression?")
				}
				c.Error(err)
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
