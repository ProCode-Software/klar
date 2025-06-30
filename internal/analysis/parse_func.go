package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) CheckFunction(
	d ast.FunctionDeclaration,
	selfType *types.Struct,
	defCtx context,
) (f types.Function) {
	f.Params = make([]types.Param, len(d.Parameters))
	ctx := runtime.NewContext(defCtx.Id)
	// Declare generic parameters
	for _, gen := range d.GenericParams {
		name := gen.Identifier
		ctx.DeclareType(name, types.Generic{Name: name}, gen.Base().Range)
	}
	for i, decParam := range d.Parameters {
		var (
			paramType   Type
			_, variadic = decParam.Type.(ast.RestType)
		)
		if variadic {
			paramType = c.ParseType(decParam.Type.(ast.RestType).Value, ctx)
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
	_ = c.CheckContext(ctx, &d.Body)
	return f
}

func (c *Checker) checkFuncDecl(decl ast.FunctionDeclaration, ctx context) {
	var (
		name = decl.Identifier
		pos  = decl.Base().Range
		f    types.Function
	)
	if decl.Struct != nil {
		// Method
		receiver := decl.Struct.(ast.TypeAlias).Identifier
		structDef, ok := ctx.TypeDeclarations[receiver]
		if !ok {
			structDef, found := ctx.ResolveType(receiver)
			if found {
				// Method outside struct scope
				c.Error(errors.ParseError{
					ErrorCode: errors.ErrMethodInOtherScope,
					Range:     pos,
					Ranges:    errors.Ranges{structDef.Position},
					Params: errors.ErrorParams{
						"name":       name,
						"structName": receiver,
					},
				})
			} else {
				c.ErrUndefinedType(receiver, decl.Struct.Base().Range, ctx)
			}
			return
		}
		if str, ok := structDef.Type.(types.Struct); ok {
			f = c.CheckFunction(decl, &str, ctx)
			str.DefineMethod(name, f, pos)
			structDef.Type = str
		} else {
			c.Error(errors.TypeError{
				ErrorCode: errors.ErrNonStructReceiver,
				Name:      receiver,
				Range:     decl.Struct.Base().Range,
				GotType:   structDef.Type,
			})
		}
		return
	}
	f = c.CheckFunction(decl, nil, ctx)
	switch err, data := ctx.DeclareFuncType(name, f, pos); err {
	case 1:
		// Overload exists
		existingOvl := data.(*types.Overload)
		c.ErrOverloadExists(name, *existingOvl, pos)
	case 2:
		// Alreay declared non-function
		c.ErrRedeclared(errors.ErrRedeclaredVar, name, pos, "function", ctx)
	}
}
