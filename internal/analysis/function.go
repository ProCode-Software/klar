package analysis

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Function represents a function type, either a declared function or a lambda.
// A Function can take multiple sets of parameters using [Overload]s.
type Function struct {
	Overloads []*Overload
	Return    Type // TODO: If returns can be Result/optional, move to Overload
	Arity     Arity
}

func (*Function) objKind() {}

// Overload represents a single overload or parameter set of a function.
// TODO: params with defaults
type Overload struct {
	*Object
	Self           *Variable // Variable's type is [*TypeName]
	Generics       []*Generic
	Params         []*Variable // Positional params
	LabelledParams []*LabelledParam
	labelMap       map[string]*Variable
	Arity          Arity
	InnerContext   *Context
	Return         Type      // Same as [Function.Return] unless this is an initializer
	NamedReturns   []*Object // Type [*Variable]
}

func (*Overload) objKind() {}

// LabelledParam represents a labelled function parameter, e.g. `label: string`.
type LabelledParam struct {
	Label string
	*Variable
}

type FunctionAlias struct {
	Target *Object // Should be [Function]
}

func (fa *FunctionAlias) Underlying() Type {
	if fa.Target == nil {
		return nil
	}
	return fa.Target.typ
}
func (*FunctionAlias) objKind() {}
func (fa *FunctionAlias) String() string {
	if fa.Target == nil {
		return "<function alias -> unknown>"
	}
	return fa.Target.String()
}

type Arity struct {
	// The minimum and maximum number of parameters the function accepts,
	// excluding labelled parametees. MaxParams can be -1 if there is no maximum.
	MinParams, MaxParams int
}

// Generic represents a generic type parameter.
type Generic struct {
	*Object
	Index int // Index within the declaration, starting at 0
}

func (g *Generic) String() string {
	if g.Object == nil {
		return "generic"
	}
	return "generic " + g.Object.name
}

// MethodAdder is implemented by types that can have methods added to them.
// Per the spec, this is implemented by [*Struct], [*Interface], and [*Enum].
type SupportsMethods interface {
	Type
	// AddMethod adds the method m to the type. If a method or field with the
	// same name already exists on the type, an error is returned. m should
	// have type [*Overload] or [*FunctionAlias].
	AddMethod(m *Object) (err *klarerrs.Error)
	GetMethods() []*Object // [*Function] or [*FunctionAlias]
}

type MethodSet struct {
	Methods      []*Object // [*Function] or [*FunctionAlias]
	methodMap    map[string]*Object
	nonMethodMap *map[string]*Object // For validating name collisions. Nil for enums.
}

const SelfName = "self"

func (c *Checker) checkFuncDecl(o *Object) {
	fn := o.typ.(*Function)
	for _, ov := range fn.Overloads {
		c.checkOverload(ov, fn)
	}
}

// fn can be nil if the overload is an initializer
func (c *Checker) checkOverload(ov *Overload, fn *Function) {
	if ov.info.funcKind != initFunc && fn == nil {
		panic("function is nil for non-initializer overload")
	}
	var (
		info   = ov.Object.info
		fctx   = ov.Object.FileContext()
		stmt   = info.node.(*ast.FunctionDeclaration)
		isInit = ov.info.funcKind == initFunc
	)
	ctx := NewContext(fctx, ov.Object.file) // Function body context
	ov.InnerContext = ctx

	// 1. Self/Receiver
	if stmt.SelfType != nil || info.receiver != nil {
		var selfPos ranges.Range
		selfName := SelfName
		if stmt.SelfName != nil {
			selfName = stmt.SelfName.Name
			selfPos = stmt.SelfName.Range()
		} else if stmt.SelfType != nil {
			selfPos = stmt.SelfType.Range()
		}
		selfObj := NewObject(selfName, ov.Object.file, selfPos, c.module, nil)
		vr := NewVariable(selfObj, SelfVar, info.receiver.TypeName())
		ov.Self = vr
		c.declare(ctx, selfObj)
	}

	// 2. Generics
	ov.Generics = c.parseGenerics(stmt.GenericParams, ov.Object.file, ctx)

	// 3. Params
	var restParam *Variable // Unlabelled
	ov.Params = make([]*Variable, 0, len(stmt.Parameters))
	ov.Arity = Arity{}
	for _, param := range stmt.Parameters {
		typ, variadic := c.parseTypeOrVariadic(param.Type, ctx)
		for _, pn := range param.Names {
			vrObj := NewObject(pn.Name.Name, ov.Object.file, pn.Name.Range(), c.module, nil)
			vr := NewVariable(vrObj, FuncParamVar, typ)
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
	if !isInit {
		fn.Arity.MinParams = min(fn.Arity.MinParams, ov.Arity.MinParams)
		if ov.Arity.MaxParams != -1 && fn.Arity.MaxParams != -1 {
			fn.Arity.MaxParams = max(fn.Arity.MaxParams, ov.Arity.MaxParams)
		}
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
		// Named returns: -> (a, b: Int)
		// Declare each key as a variable
		if tuple, ok := stmt.ReturnType.(*ast.TupleType); ok {
			retTuple := make(Tuple, 0, len(tuple.Values))
			for _, pair := range tuple.Values {
				typ := c.parseType(pair.Value, ctx)
				for _, key := range pair.Keys {
					retTuple = append(retTuple, typ)
					obj := NewObject(
						key.Name,
						ov.Object.file, key.Range(), ov.Object.module, nil,
					)
					_ = NewVariable(obj, LocalVar, typ)
					c.declare(ctx, obj)
					ov.NamedReturns = append(ov.NamedReturns, obj)
				}
			}
			ret = retTuple
		} else {
			ret = c.parseType(stmt.ReturnType, ctx)
		}
	}
	if !isInit && fn.Return != nil && ret != fn.Return {
		// All overloads must have the same return type
		// TODO: use a compatibility check instead of !=
		// TODO: hint for Nothing != ()
	} else if !isInit {
		fn.Return = ret
	}

	// 5. Body
	if !c.Options.IgnoreFuncBodies && (stmt.Body != nil || stmt.Expression != nil) {
		c.queue(func() { c.checkFuncBody(stmt, ov, fn, fctx) }, true)
	}
}

// fn could be nil if the overload is an initializer
func (c *Checker) checkFuncBody(stmt *ast.FunctionDeclaration, ov *Overload,
	fn *Function, fctx *Context,
) {
	// TODO: Extract some fields from [Checker] such as moduleDecls for
	// use in nested contexts.
	ctx := ov.InnerContext
	if stmt.Body == nil {
		// Function expression
		return
	}
	sctx := newStmtContext(ctx, ov.file, allowReturn)
	c.checkBlock(stmt.Body.Body, sctx)
}

func (c *Checker) parseGenerics(names []ast.Identifier,
	fid FileID, ctx *Context,
) []*Generic {
	generics := make([]*Generic, len(names))
	for i, param := range names {
		genObj := NewObject(param.Name, fid, param.Range(), c.module, &TypeName{Name: param.Name})
		gen := newGeneric(genObj, i)
		c.declare(ctx, genObj)
		generics[i] = gen
	}
	return generics
}

func newGeneric(o *Object, index int) *Generic {
	gen := &Generic{Object: o, Index: index}
	o.TypeName().Type = gen
	return gen
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
		if ret := fn.Return.Kind(); ret != NothingType && ret != InvalidType {
			b.WriteString(" -> ")
			b.WriteString(fn.Return.String())
		}
	}
	return b.String()
}

func (fn *Function) Underlying() Type {
	// If Return == nil, the function is incomplete
	if fn.Return == nil {
		return nil
	}
	return fn
}

func (o *Overload) Underlying() Type {
	if o.InnerContext == nil {
		return nil
	}
	return o
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
			b.WriteString(g.name)
		}
		b.WriteByte('>')
	}
	// Params
	b.WriteByte('(')
	for i, param := range o.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.Type.String())
	}
	// Labelled params
	for i, param := range o.LabelledParams {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.Label)
		b.WriteString(": ")
		b.WriteString(param.Type.String())
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
		return m.defineNewMethod(obj) // New method
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
		err.Name = obj.name
		err.AddDetail(
			"Other definition of "+klarerrs.Quote(obj.name),
			existing.FilePath(), existing.rang,
		)
		err.Hint("An alias can't be used as an overload")
		return err
	}
	panic("unreachable")
}

func (m *MethodSet) defineNewMethod(obj *Object) (err *klarerrs.Error) {
	// Wrap the possible overload in a Function
	if ov, ok := obj.typ.(*Overload); ok {
		obj = NewObject(obj.name, obj.file, obj.rang, obj.module, &Function{
			Overloads: []*Overload{ov},
		})
	}
	m.methodMap[obj.name] = obj
	m.Methods = append(m.Methods, obj)

	if m.nonMethodMap == nil {
		return nil
	}
	// Check if a method shares the same name as something else (such as a field
	// for structs)
	if *m.nonMethodMap != nil {
		if existing, ok := (*m.nonMethodMap)[obj.name]; ok {
			err := klarerrs.Range(klarerrs.ErrFieldAndMethodSameName, obj.rang)
			err.Label = "There is also a field named " + quote(obj.name)
			err.Name = obj.name
			err.AddDetail(
				"The conflicting field was defined here",
				existing.FilePath(), existing.rang,
			)
			return err
		}
	} else {
		*m.nonMethodMap = make(map[string]*Object)
	}
	// Add the method to the map of both fields and methods. Structs that
	// embed [MethodSet] will use this map for indexing.
	(*m.nonMethodMap)[obj.name] = obj
	return nil
}

func (m *MethodSet) GetMethods() []*Object { return m.Methods }
