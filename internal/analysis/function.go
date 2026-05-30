package analysis

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// Function represents a function type, either a declared function or a lambda.
// A Function can take multiple sets of parameters using [Overload]s.
type Function struct {
	Overloads []*Overload
	Return    Type
	Arity     Arity
}

// Overload represents a single overload or parameter set of a function.
// TODO: params with defaults
type Overload struct {
	*Object
	Self           *Variable
	Generics       []*Generic
	Params         []*Variable
	LabelledParams []*LabelledParam
	labelMap       map[string]*Variable
	Arity          Arity
	InnerContext   *Context
}

// LabelledParam represents a labelled function parameter, e.g. `label: string`.
type LabelledParam struct {
	Label string
	*Variable
}

type FunctionAlias struct {
	Target *Object // Should be [Function]
}

type Arity struct {
	// The minimum and maximum number of parameters the function accepts,
	// excluding labelled parametees. MaxParams can be -1 if there is no maximum.
	MinParams, MaxParams int
}

// Generic represents a generic type parameter.
type Generic struct {
	*Object
	Name string
}

// MethodAdder is implemented by types that can have methods added to them.
// Per the spec, this is implemented by [*Struct], [*Interface], and [*Enum].
type SupportsMethods interface {
	// AddMethod adds the method m to the type. If a method or field with the
	// same name already exists on the type, an error is returned. m should
	// have type [*Overload] or [*FunctionAlias].
	AddMethod(m *Object) (err *klarerrs.Error)
}

type MethodSet struct {
	Methods      []*Object // [*Function] or [*FunctionAlias]
	methodMap    map[string]*Object
	nonMethodMap *map[string]*Object // For validating name collisions. Nil for enums.
}

func (c *Checker) checkFuncDecl(o *Object) {
	fn := o.typ.(*Function)
	for _, ov := range fn.Overloads {
		ovInfo := c.moduleDecls[ov.Object]
		stmt := ovInfo.node.(*ast.FunctionDeclaration)
		ctx := NewContext(o.context, o.file) // Function body context

		// 1. Self/Receiver
		if stmt.SelfType != nil {
			selfName := "self"
			selfPos := stmt.SelfType.Range()
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
		var restParam *Variable // Unlabelled
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
						if restParam != nil {
							// Variadic exists
						}
						restParam = vr
					} else {
						optional := false // TODO: check if typ is optional
						if !optional {
							ov.Arity.MinParams++
						}
						ov.Arity.MaxParams++
					}
				} else {
					// Labelled param
					lp := &LabelledParam{pn.Label.Name, vr}
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
		// Verify that the variadic param is the last unlabelled param
		if restParam != nil && ov.Params[len(ov.Params)-1] != restParam {
		}

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
			c.queue(func() { c.checkFuncBody(stmt, fn, ov) }, beforeFinish)
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

// parseTypeOrVariadic parses [*ast.RestType], returning a [*List]. If t is
// not [*ast.RestType], parseTypeOrVariadic is the same as [Checker.parseType].
// This should be the only function that accepts variadic types.
func (c *Checker) parseTypeOrVariadic(t ast.Type, ctx *Context) (typ Type, variadic bool) {
	if dt, ok := t.(*ast.RestType); ok {
		return &List{c.parseType(dt.Value, ctx)}, true
	}
	return c.parseType(t, ctx), false
}

func (fn *Function) Kind() Kind     { return KindFunction }
func (fn *Function) String() string { return fn.StringWithName("") }
func (fn *Function) StringWithName(name string) string {
	var b strings.Builder
	b.WriteString("func")
	if name != "" {
		b.WriteByte(' ')
		b.WriteString(name)
	}
	if len(fn.Overloads) == 1 {
		b.WriteString(fn.Overloads[0].String())
	}
	if fn.Return != nil {
		switch fn.Return.Kind() {
		case NothingType, InvalidType, KindUnreachable:
		default:
			b.WriteString(" -> ")
			b.WriteString(TypeToString(fn.Return))
		}
	}
	return b.String()
}

func (o *Overload) Kind() Kind { return KindFunction }

func (o *Overload) String() string {
	var b strings.Builder
	// Generics
	if len(o.Generics) > 0 {
		b.WriteByte('<')
		for i, g := range o.Generics {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(g.Name)
		}
		b.WriteByte('>')
	}
	// Params
	b.WriteByte('(')
	for i, param := range o.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.String())
	}
	// Labelled params
	for i, param := range o.LabelledParams {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.String())
	}
	b.WriteByte(')')
	return b.String()
}

func (g *Generic) Kind() Kind { return KindGeneric }

func (a *FunctionAlias) Kind() Kind { return KindFunction }

func (m *MethodSet) AddMethod(obj *Object) (err *klarerrs.Error) {
	if m.methodMap == nil {
		m.methodMap = make(map[string]*Object)
	}
	var funcAndAliasConflict bool
	existing, ok := m.methodMap[obj.name]
	if !ok {
		// New method
		//
		// Check if a method shares the same name as something else (such as a field
		// for structs)
		if m.nonMethodMap != nil && *m.nonMethodMap != nil {
			if _, ok := (*m.nonMethodMap)[obj.name]; ok {
				return nil
			}
		}
		if ov, ok := obj.typ.(*Overload); ok {
			obj = NewObject(obj.name, obj.file, obj.rang, obj.module, &Function{
				Overloads: []*Overload{ov},
			})
		}
		m.methodMap[obj.name] = obj
		m.Methods = append(m.Methods, obj)
		return nil
	}

	switch old := existing.typ.(type) {
	case *Function:
		if _, ok := obj.typ.(*FunctionAlias); ok {
			funcAndAliasConflict = true
			break
		}
		// Add overload to existing function
		old.Overloads = append(old.Overloads, obj.typ.(*Overload))
		m.Methods = append(m.Methods, obj)
		return nil
	case *FunctionAlias:
		if _, ok := obj.typ.(*Function); ok {
			funcAndAliasConflict = true // Just for a better error
			break
		}
		// Two aliases with the same name
		return redeclaredError(obj, existing, false)
	default:
		panic(fmt.Sprintf("method should be *Function or *FunctionAlias: found %T", old))
	}
	// Report the error
	if funcAndAliasConflict {
		err := klarerrs.Range(klarerrs.ErrAliasAndMethodSameName, obj.rang)
		err.AddDetail(
			"Other definition of "+klarerrs.Quote(obj.name),
			existing.FilePath(), existing.rang,
		)
		err.Hint("An alias can't be used as an overload")
		return err
	}
	panic("unreachable")
}
