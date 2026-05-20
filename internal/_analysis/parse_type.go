package analysis

import (
	"fmt"
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

func (c *Checker) validateVariadicParam(index int, len int, typ ast.Type) {
	if index != len-1 {
		c.Error(errors.Range(errors.ErrVariadicLast, typ.GetRange()))
	}
}

func (c *Checker) ParseType(t ast.Type, ctx context) Type {
	if t == nil {
		panic("ParseType: nil type")
	}
	switch t := t.(type) {
	case *ast.TypeAlias:
		name := t.Identifier
		if decl, ok := ctx.ResolveType(name); !ok {
			c.ErrUndefinedType(name, t.Range, ctx)
			return types.InvalidType
		} else {
			return types.Ref{name, &decl.Type}
		}
	case *ast.FunctionType:
		var f types.Lambda
		f.Params = make([]types.Param, len(t.Parameters.Values))
		/* for i, paramType := range t.Parameters {
			var typ Type
			_, variadic := paramType.(*ast.RestType)
			if variadic {
				typ = c.ParseType(paramType.(*ast.RestType).Value, ctx)
				c.validateVariadicParam(i, len(t.Parameters), paramType)
			} else {
				typ = c.ParseType(paramType, ctx)
			}
			f.Params[i] = types.Param{Type: typ, Variadic: variadic}
		} */
		f.Return = c.ParseType(t.ReturnType, ctx)
		return f
	case *ast.PrimitiveType:
		return types.PrimitiveMap[t.Primitive]
	case *ast.UnionType:
		items := make([]Type, len(t.Options))
		for i, opt := range t.Options {
			items[i] = c.ParseType(opt, ctx)
		}
		return types.Union{Options: items}
	case *ast.OptionalType:
		return types.Optional{Underlying: c.ParseType(t.Value, ctx)}
	case *ast.BadExpression:
		return types.InvalidType
	case *ast.ListType:
		return types.List{Of: c.ParseType(t.Value, ctx)}
	case *ast.GenericType:
		// Must be Result or Map
		var invalid bool
		prim, isPrim := t.Name.(*ast.PrimitiveType)
		parseWithDefault := func(index int, def Type) Type {
			if index >= len(t.Parameters) {
				return def
			}
			return c.ParseType(t.Parameters[index], ctx)
		}
		validateLen := func(min, max int) {
			got := len(t.Parameters)
			if got >= min && got <= max {
				return
			}
			c.Error(errors.Range(
				errors.ErrWrongTypeParamLen, t.GetRange(),
				errors.ErrorParams{"min": min, "max": max, "got": got},
			))
		}
		switch prim.Primitive {
		case ast.PrimitiveResult:
			validateLen(0, 2)
			return types.Result{
				SuccessType: parseWithDefault(0, types.Nothing),
				FailureType: parseWithDefault(1, types.Error),
			}
		case ast.PrimitiveMap:
			validateLen(0, 2)
			return types.Map{
				KeyType:   parseWithDefault(0, types.Any),
				ValueType: parseWithDefault(1, types.Any),
			}
		default:
			invalid = true
		}
		if invalid || !isPrim {
			err := errors.TypeError{
				Code: errors.ErrNoGenerics,
				Range:     t.GetRange(),
				Params:    errors.ErrorParams{"type": c.ParseType(t.Name, ctx)},
			}
			err.Hint("Only 'Map' and 'Result' types are generic")
			c.Error(err)
		}
	case *ast.RestType:
		c.Error(errors.Range(errors.ErrInvalidRestType, t.GetRange()))
		return types.InvalidType
	case *ast.TupleType:
		items := make([]Type, len(t.Values))
		for i, item := range t.Values {
			items[i] = c.ParseType(item, ctx)
		}
		return types.Tuple{Items: items}
	default:
		panic(fmt.Sprintf("ParseType: unknown type: %T", t))
	}
	return types.InvalidType
}

func (c *Checker) parseInheritance(
	s *types.Struct, inheritedTypes []ast.Type, ctx context, isIntf bool,
) {
	var implements []*runtime.TypeDeclaration
	for _, item := range inheritedTypes {
		var name string
		switch item := item.(type) {
		case *ast.TypeAlias:
			name = item.Identifier
		case *ast.PrimitiveType:
			// TODO: inheriting primitive types
			continue
		}
		var (
			decl, _   = ctx.ResolveType(name)
			inherited types.HasFields
		)
		if _, ok := decl.Type.(types.Interface); ok && !isIntf {
			implements = append(implements, decl)
			continue
		}
		switch typeVal := decl.Type.(type) {
		case types.HasFields:
			inherited = typeVal
		default:
			continue
			// Can inherit other types
			/* err := errors.NamedTypeError(
				errors.ErrInheritNonStructOrIntf, name, item.GetRange(),
			)
			err.SetParam("type", decl.Type)
			c.Error(err)
			continue */
		}
		for k, v := range inherited.GetFields() {
			if _, ok := s.Fields[k]; ok && !isIntf {
				c.Error(errors.NamedTypeError(
					errors.ErrConflictingInherit, k, item.GetRange(),
				))
				continue
			}
			s.Fields[k] = v
			s.Order = append(s.Order, k)
		}
		for fnName, allOvlds := range inherited.GetMethods() {
			for _, ovl := range allOvlds {
				if _, exists := s.Methods[fnName].Get(ovl.Params); exists && !isIntf {
					err := errors.Range(
						errors.ErrConflictingInherit, item.GetRange(),
						errors.ErrorParams{"overload": &ovl},
					)
					err.Name = fnName
					c.Error(err)
					continue
				}
				s.Methods[fnName] = append(s.Methods[fnName], ovl)
			}
		}
	}
	s.Implements = implements
}

func (c *Checker) ParseStruct(d *ast.StructDeclaration, ctx context) (s types.Struct) {
	s.Fields = make(map[string]Type, len(d.Fields))
	s.Methods = make(map[string]types.Overloads)
	c.parseInheritance(&s, d.InheritedTypes, ctx, false)
	s.Order = slices.Grow(s.Order, len(d.Fields))
	for _, field := range d.Fields {
		s.Order = append(s.Order, field.Identifier)
		s.Fields[field.Identifier] = c.ParseType(field.Type, ctx)
	}
	return s
}

func (c *Checker) parseIntfMethod(meth *ast.MethodType, ctx context) (f types.Function) {
	f.Params = make([]types.Param, len(meth.Parameters))
	for i, param := range meth.Parameters {
		var typ Type
		_, variadic := param.Type.(*ast.RestType)
		if variadic {
			typ = c.ParseType(param.Type.(*ast.RestType).Value, ctx)
		} else {
			typ = c.ParseType(param.Type, ctx)
		}
		c.validateVariadicParam(i, len(meth.Parameters), param.Type)
		f.Params = append(f.Params, types.Param{
			Label:    param.Label,
			Type:     typ,
			Variadic: variadic,
		})
	}
	return f
}

func (c *Checker) ParseInterface(
	d *ast.InterfaceDeclaration, ctx context,
) (i types.Interface) {
	tmpStruct := types.Struct{
		Fields:  make(map[string]Type, len(d.Fields)),
		Methods: make(map[string]types.Overloads),
	}
	c.parseInheritance(&tmpStruct, d.InheritedTypes, ctx, true)
	i.Fields, i.Methods = tmpStruct.Fields, tmpStruct.Methods
	for _, field := range d.Fields {
		if val, ok := field.Value.(*ast.MethodType); ok {
			i.Methods[field.Key] = append(
				i.Methods[field.Key], types.Overload{
					Function: c.parseIntfMethod(val, ctx),
					Position: val.GetRange(),
				},
			)
			continue
		}
		i.Fields[field.Key] = c.ParseType(field.Value, ctx)
	}
	return i
}
