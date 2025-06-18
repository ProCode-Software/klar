package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) ParseFunction(d ast.FunctionDeclaration, ctx context) (f types.Function) {
	f.Params = make([]types.Param, len(d.Parameters))
	for i, decParam := range d.Parameters {
		var (
			paramType   Type
			_, variadic = decParam.Type.(ast.RestType)
		)
		if variadic {
			paramType = c.ParseType(decParam.Type.(ast.RestType).Value, ctx)
			c.validateVariadicParam(i, len(d.Parameters), decParam.Type)
		} else if decParam.Type == nil {
			paramType = types.InvalidType
			// Syntax error if type not provided
		} else {
			paramType = c.ParseType(decParam.Type, ctx)
		}
		f.Params[i] = types.Param{
			Label:    decParam.Label,
			Variadic: variadic,
			Type:     paramType,
		}
	}
	// Nil if inferred
	if d.ReturnType != nil {
		f.Return = c.ParseType(d.ReturnType, ctx)
	}
	return f
}

func (c *Checker) parseFuncDecls(funcs []ast.FunctionDeclaration, ctx context) {
	for _, decl := range funcs {
		var (
			name = decl.Identifier
			pos  = decl.Base().Range
			f    = c.ParseFunction(decl, ctx)
		)
		if decl.Struct != nil {
			// Method
			receiver := decl.Struct.(ast.TypeAlias).Identifier
			structDef, found := ctx.ResolveType(receiver)
			if !found {
				c.undefinedType(receiver, decl.Struct.Base().Range, ctx)
				continue
			}
			if str, ok := structDef.Type.(types.Struct); ok {
				str.DefineMethod(name, f, pos)
				structDef.Type = str
			} else {
				err := errors.TypeMismatch(nil, structDef.Type, decl.Struct.Base().Range)
				err.ErrorCode = errors.ErrNonStructReceiver
				err.Name = receiver
				c.Error(err)
			}
			continue
		}
		switch err, data := ctx.DeclareFuncType(name, f, pos); err {
		case 1:
			// Overload exists
			existingOvl := data.(types.Overload)
			c.errOverloadExists(name, existingOvl, pos)
		case 2:
			// Alreay declared non-function
			c.errRedeclared(errors.ErrRedeclaredVar, name, pos, "function", ctx)
		}
	}
}
