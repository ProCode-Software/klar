package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ast/typed"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) CheckFunction(
	d *ast.FunctionDeclaration,
	selfType *types.Struct,
	defCtx context,
) (f types.Function) {
	f.Params = make([]types.Param, len(d.Parameters))
	ctx := runtime.NewContext(defCtx.Id)
	// Declare generic parameters
	for _, gen := range d.GenericParams {
		name := gen.Identifier
		ctx.DeclareType(name, types.Generic{Name: name}, gen.GetRange())
	}
	for i, decParam := range d.Parameters {
		var (
			paramType   Type
			_, variadic = decParam.Type.(*ast.RestType)
		)
		if variadic {
			paramType = c.ParseType(decParam.Type.(*ast.RestType).Value, ctx)
			c.validateVariadicParam(i, len(d.Parameters), decParam.Type)
		} else if decParam.Type == nil {
			paramType = types.InvalidType // Syntax error if type not provided
		} else {
			paramType = c.ParseType(decParam.Type, ctx)
		}
		// TODO: check type of default value
		f.Params[i] = types.Param{
			Label:    decParam.Label,
			Variadic: variadic,
			Type:     paramType,
		}
	}
	// Nil if inferred
	inferReturn := d.ReturnType == nil
	if !inferReturn {
		f.Return = c.ParseType(d.ReturnType, ctx)
	}
	// Self variable
	if selfType != nil {
		ctx.Declare("self", true, selfType, defaultRange) // todo: check if pointer selfType should be used
	}
	// Check statements
	_, _ = c.CheckContext(ctx, d.Body)
	return f
}

func (c *Checker) checkFuncDecl(decl *ast.FunctionDeclaration, ctx context) *typed.FunctionDecl {
	var (
		name = decl.Identifier
		pos  = decl.GetRange()
		f    types.Function
	)
	if decl.Struct != nil {
		// Method
		receiver := decl.Struct.(*ast.TypeAlias).Identifier
		structDef, ok := ctx.TypeDeclarations[receiver]
		if !ok {
			structDef, found := ctx.ResolveType(receiver)
			if found {
				// Method outside struct scope
				// other: structDef.Position
				c.Error(klarerrs.Error{
					Code: errors.ErrMethodInOtherScope,
					Details: []errors.Detail{{
						Range:       structDef.Position,
						Description: fmt.Sprintf("%s was declared here", errors.Quote(receiver)),
					}},
					Params: errors.ErrorParams{
						"name":       name,
						"structName": receiver,
					},
				})
			} else {
				c.ErrUndefinedType(receiver, decl.Struct.GetRange(), ctx)
			}
			return nil
		}
		if str, ok := structDef.Type.(types.Struct); ok {
			f = c.CheckFunction(decl, &str, ctx)
			str.DefineMethod(name.Identifier, f, pos)
			structDef.Type = str
		} else {
			c.Error(errors.TypeError{
				Code:    errors.ErrNonStructReceiver,
				Name:    receiver,
				Range:   decl.Struct.GetRange(),
				GotType: structDef.Type,
			})
		}
		return nil
	}
	f = c.CheckFunction(decl, nil, ctx)
	switch err, data := ctx.DeclareFuncType(name.Identifier, f, pos); err {
	case 1:
		// Overload exists
		existingOvl := data.(*types.Overload)
		c.ErrOverloadExists(name.Identifier, *existingOvl, pos)
	case 2:
		// Alreay declared non-function
		c.ErrRedeclared(errors.ErrRedeclaredVar, name.Identifier, pos, "function", ctx)
	}
	return nil
}
