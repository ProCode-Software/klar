package analysis

import "github.com/ProCode-Software/klar/internal/ast"

func (c *Checker) checkFuncDecl(o *Object) {
	fn := o.typ.(*Function)
	for _, ov := range fn.Overloads {
		ovInfo := c.moduleDecls[ov.Object]
		stmt := ovInfo.node.(*ast.FunctionDeclaration)
		ctx := NewContext(o.context) // Function body context

		// 1. Self/Receiver
		if stmt.Struct != nil {
			selfName := "self"
			selfPos := stmt.Struct.Range()
			if stmt.SelfName != nil {
				selfName = stmt.SelfName.Name
				selfPos = stmt.SelfName.Range()
			}
			self := &Variable{VarKind: SelfVar}
			selfObj := NewObject(selfName, ov.Object.file, selfPos, c.module, self)
			self.Object = selfObj
			ov.Self = self
			c.declare(ctx, selfObj)
		}

		// 2. Generics
		ov.Generics = c.parseGenerics(stmt.GenericParams, ov.Object.file, ctx)

		// 3. Params
		ov.Params = make([]*Variable, 0, len(stmt.Parameters))
		ov.Arity = Arity{}
		for _, param := range stmt.Parameters {
			typ, variadic := c.parseTypeOrVariadic(param.Type, o.context)
			for _, pn := range param.Names {
				vr := &Variable{VarKind: FuncParamVar, Type: typ}
				vrObj := NewObject(pn.Name.Name, ov.Object.file, pn.Name.Range(), c.module, vr)
				vr.Object = vrObj
				if variadic {
					vr.Object.flags |= VariadicParam
				}

				if pn.Label.IsZero() {
					// Normal param
					ov.Params = append(ov.Params, vr)

					// Adjust arity: Arity only counts unlabelled params
					if variadic {
						// If there is a variadic parameter, there is no max number of params
						ov.Arity.MaxParams = -1
						fn.Arity.MaxParams = -1
					} else {
						optional := false // TODO: check if typ is optional
						if !optional {
							ov.Arity.MinParams++
						}
						ov.Arity.MaxParams++
					}
				} else {
					// Labelled param
					lp := &LabelledParam{
						Label:    pn.Label.Name,
						Variable: vr,
					}
					ov.LabelledParams = append(ov.LabelledParams, lp)
					if ov.labelMap == nil {
						ov.labelMap = make(map[string]*Variable)
					}
					ov.labelMap[pn.Label.Name] = vr
				}
				c.declare(ctx, vrObj)
				_ = param.Default // TODO
			}
		}
		// Set the arity bounds for the whole function
		fn.Arity.MinParams = min(fn.Arity.MinParams, ov.Arity.MinParams)
		if ov.Arity.MaxParams != -1 && fn.Arity.MaxParams != -1 {
			fn.Arity.MaxParams = max(fn.Arity.MaxParams, ov.Arity.MaxParams)
		}
		// TODO: verify that the variadic param is the last unlabelled param

		// 4. Return type
		var ret Type
		if stmt.ReturnType == nil {
			// No explicit return type = Nothing
			ret = NothingType
		} else {
			// TODO: in the context of generics
			ret = c.parseType(stmt.ReturnType, o.context)
		}
		if fn.Return != nil && ret != fn.Return {
			// All overloads must have the same return type
			// TODO: use a compatibility check instead of !=
			// TODO: hint for Nothing != ()
		} else {
			fn.Return = ret
		}

		// 5. Body
		if !c.Options.IgnoreFuncBodies {
			c.queue(func() { c.checkFuncBody(stmt, fn, ov) })
		}
	}
}

func (c *Checker) checkFuncBody(stmt *ast.FunctionDeclaration, fn *Function, ov *Overload) {
	_ = ov.InnerContext
}

func (c *Checker) parseGenerics(names []ast.Identifier,
	fid FileID, ctx *Context,
) []*Generic {
	generics := make([]*Generic, len(names))
	for i, param := range names {
		gen := &Generic{Name: param.Name}
		genObj := NewObject(param.Name, fid, param.Range(), c.module, gen)
		gen.Object = genObj
		c.declare(ctx, genObj)
		generics[i] = gen
	}
	return generics
}

func (c *Checker) parseTypeOrVariadic(t ast.Type, ctx *Context) (typ Type, variadic bool) {
	if dt, ok := t.(*ast.RestType); ok {
		return &List{c.parseType(dt.Value, ctx)}, true
	}
	return c.parseType(t, ctx), false
}
