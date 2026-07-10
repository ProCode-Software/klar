package analysis

import (
	"fmt"
	"maps"
	"slices"
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

func (a Arity) InRange(n int) bool {
	return a.MinParams <= n && (a.MaxParams == -1 || n <= a.MaxParams)
}

// Generic represents a generic type parameter.
type Generic struct {
	*Object
	Index int // Index within the declaration, starting at 0
}

func (g *Generic) Underlying() Type { return g }

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
		c.checkOverload(ov, o)
	}
	// Ensure no overloads are ambiguous
	fn.Overloads = c.checkOverloadAmbiguity(fn.Overloads)
}

// fn can be nil if the overload is an initializer
func (c *Checker) checkOverload(ov *Overload, fnObj *Object) {
	var (
		info   = ov.Object.info
		fctx   = ov.Object.LookupContext()
		stmt   = info.node.(*ast.FunctionDeclaration)
		isInit = info.funcKind == initFunc
		fn     *Function
	)
	if fnObj == nil && !isInit {
		panic("function is nil for non-initializer overload")
	}
	if fnObj != nil {
		fn = fnObj.typ.(*Function)
	}
	ctx := NewContext(fctx, ov.Object.file) // Function body context
	ov.InnerContext = ctx

	// 1. Self/Receiver
	if stmt.SelfType != nil || info.receiver != nil {
		var selfPos ranges.Range
		selfName := SelfName
		switch {
		case stmt.SelfName != nil: // Method with explicit self alias
			selfName, selfPos = stmt.SelfName.Name, stmt.SelfName.Range()
		case stmt.SelfType != nil: // Method
			selfPos = stmt.SelfType.Range()
		case isInit: // Initializer
			selfPos = stmt.Identifier.Range()
		}
		selfObj := NewObject(selfName, ov.Object.file, selfPos, c.module, nil)
		tn := info.receiver.TypeName()
		if c.module.Flags.Has(BootstrapModule) {
			// TODO: This is a temporary solution
			tn = c.wrapBootstrappedTypeName(tn, info.receiver)
		}
		ov.Self = NewVariable(selfObj, SelfVar, tn)
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
			c.declare(ctx, vrObj)
			switch {
			case !pn.Label.IsZero():
				// Labelled param
				lp := &LabelledParam{pn.Label.Name, vr}
				ov.LabelledParams = append(ov.LabelledParams, lp)
				if ov.labelMap == nil {
					ov.labelMap = make(map[string]*Variable)
				}
				// TODO: Check for name conflicts
				ov.labelMap[pn.Label.Name] = vr
			case variadic:
				// Unlabelled variadic param
				ov.Params = append(ov.Params, vr)

				// If there is a variadic parameter, there is no max number of params
				ov.Arity.MaxParams = -1
				if !isInit {
					fn.Arity.MaxParams = -1
				}
				// Ensure there is only 1 variadic param in the overload
				if restParam != nil {
					// Variadic exists
					err := objectError(klarerrs.ErrMultipleVariadicParam, vrObj)
					err.Label = "A variadic parameter was already defined"
					err.AddHighlight(
						"The first variadic parameter was defined here",
						restParam.Object.rang,
					)
					c.fileError(err, ov.file)
					break
				}
				restParam = vr
			default:
				// Normal param
				ov.Params = append(ov.Params, vr)
				// Adjust arity: Arity only counts unlabelled params
				optional := typ.Kind() == KindOptional // TODO: this doesn't handle unions
				if !optional {
					ov.Arity.MinParams++
				}
				ov.Arity.MaxParams++
			}

			// Check default value
			if param.Default != nil {
				// A variadic parameter can't have a default value
				// 	func _(items: ...Int = [1, 2, 3])
				if variadic {
					err := klarerrs.Node(klarerrs.ErrVariadicDefault, param.Default)
					err.Label = "Remove this default value"
					err.AddHighlight(
						"This parameter is defined as variadic",
						param.Type.GetRange(),
					)
					c.fileError(err, ov.file)
					continue
				}
				// TODO: Should it be delayed?
				// TODO: Should a default value be allowed with a generic param?
				t := NewExpr(ctx, constExpr)
				c.checkExpr(param.Default, t)
				if !Compatible(t.Type, typ) {
					err := typeMismatch(typ, t.Type, param.Default.GetRange())
					err.Node = param.Default
					err.AddHighlight(
						"The type of the parameter is "+quoteAka(typ),
						param.Type.GetRange(),
					)
				}
			}
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
		err := objectError(klarerrs.ErrVariadicNotLast, restParam.Object)
		err.Label = "This should be the last unlabelled parameter"
		// Highlight the params after this
		after := ov.Params[slices.Index(ov.Params, restParam)+1:]
		r := ranges.Range{after[0].Object.rang.Start, after[len(after)-1].Object.rang.End}
		err.AddHighlight("It should go after these", r)
		c.fileError(err, restParam.Object.file)
	}

	// 4. Body expression, which may be used to infer return type
	var bodyExpr *Expr
	if !c.Options.IgnoreFuncBodies && stmt.Expression != nil {
		bodyExpr = c.checkExpr(stmt.Expression, NewExpr(ctx, 0))
	}

	// 5. Return type
	var retRange ranges.Range
	switch rt := stmt.ReturnType.(type) {
	default:
		ov.Return = c.parseType(stmt.ReturnType, ctx)
		retRange = stmt.ReturnType.GetRange()
	case nil:
		switch {
		case bodyExpr != nil:
			// Inferred from body expression
			ov.Return, retRange = bodyExpr.Type, stmt.Expression.GetRange()
		case isInit:
			// `func Int()` implicitly returns Int
			ov.Return, retRange = info.receiver.TypeName(), stmt.Identifier.Range()
		default:
			ov.Return, retRange = NothingType, stmt.Range
		}
	case *ast.TupleType:
		// Named returns: -> (a, b: Int)
		// Declare each key as a variable. If any are present, an explicit 'return'
		// statement is optional within the body (unlike Go).
		//
		// Discard keys don't count as named, so if the return type is a tuple
		// with all discard keys, there are no named returns, and an explicit
		// 'return' statement is required.
		retTuple := make(Tuple, 0, len(rt.Values))
		for _, pair := range rt.Values {
			typ := c.parseType(pair.Value, ctx)
			for _, key := range pair.Keys {
				retTuple = append(retTuple, typ)
				if key.IsZero() || key.IsDiscard() {
					continue
				}
				vr := NewObject(
					key.Name,
					ov.Object.file, key.Range(), ov.Object.module, nil,
				)
				vr.flags |= needsSet
				_ = NewVariable(vr, LocalVar, typ)
				c.declare(ctx, vr)
				ov.NamedReturns = append(ov.NamedReturns, vr)
			}
			if len(pair.Keys) == 0 {
				// Otherwise the type won't be appended if there are no keys
				retTuple = append(retTuple, typ)
			}
		}
		ov.Return, retRange = retTuple, rt.Range
	case *ast.ParenType:
		// Similar to tuple, but has only 1 item
		ov.Return, retRange = c.parseType(rt.Type, ctx), rt.Range
		if rt.Label.IsZero() || rt.Label.IsDiscard() {
			return
		}
		vr := NewObject(
			rt.Label.Name,
			ov.Object.file, rt.Label.Range(), ov.Object.module, nil,
		)
		vr.flags |= needsSet
		_ = NewVariable(vr, LocalVar, ov.Return)
		c.declare(ctx, vr)
		ov.NamedReturns = append(ov.NamedReturns, vr)
	}

	// Ensure the return type is the same across all overloads. This
	// isn't checked for initializers, where an initializer for T can
	// return T, Result<T>, or T?.
	//
	//   func Int(float: Float) -> Int
	//   func Int(str: String) -> Result<Int>
	//
	// Equality of return types are strict, so this will fail:
	// 	type #Tag
	// 	type Impl: Tag
	// 	func x() -> Tag
	//  func x() -> Impl
	switch {
	case isInit:
		// Change return type of `Result` (exact syntax) or `Result?` to
		// `Result<T>` from `Result<Nothing>`. We're intentionally
		// checking for equality by reference.
		var changeFromResultNothing func(*Type)
		changeFromResultNothing = func(typ *Type) {
			switch ret := ov.Return.(type) {
			case *Optional:
				changeFromResultNothing(&ret.Elem)
			case *Result:
				if *typ == ResultNothing {
					ov.Return = info.receiver.typ
				}
			}
		}
		if info.receiver.name != "List" {
			// Don't change the return type of List initializers
			//	func List(...) -> [Result] should return [Result<Nothing>]
			changeFromResultNothing(&ov.Return)
		}

		switch {
		// An initializer named 'List' must return a list (or a list as an optional/result)
		case info.receiver.name == "List":
			if ConcreteTypeOf(ov.Return).Kind() != KindList {
				err := klarerrs.Range(klarerrs.ErrInvalidListInitReturn, retRange)
				c.fileError(err, ov.file)
				ov.Return = &List{InvalidType}
			}

		// Check that the overload's concrete type is the one it initializes
		case !TypesEqual(ConcreteTypeOf(ov.Return), info.receiver.typ):
			err := klarerrs.Range(klarerrs.ErrInvalidInitReturn, retRange)
			err.Name = ov.name
			// Show a hint for `func T() -> [T]`
			if asList := (&List{info.receiver.typ}); TypesEqual(ov.Return, asList) {
				err.Label = "An initializer can't return a list of " +
					quote(asList.String())
			}
			err.AddHighlight("This type is being initialized", stmt.Identifier.Range())
			c.fileError(err, ov.file)
			ov.Return = info.receiver.typ
		}
	case fn.Return == nil:
		fn.Return = ov.Return
	case !TypesEqual(ov.Return, fn.Return):
		err := typeMismatch(fn.Return, ov.Return, retRange)
		err.Code = klarerrs.ErrOverloadReturnMismatch
		err.Name = fnObj.name
		err.Label = "This should return " + quoteAka(fn.Return)
		err.AddDetail(
			"The return type was defined with the first overload here",
			fnObj.FilePath(), fnObj.rang,
		)
		c.fileError(err, ov.file)
	default: // Correct return types
	}

	// 6. Body
	if !c.Options.IgnoreFuncBodies && stmt.Body != nil {
		c.queue(func() { c.checkFuncBody(stmt, ov, fn, retRange, fctx) }, true)
	} else if bodyExpr != nil {
		// This is queued because a function body's returns are also queued.
		// TODO: This is for consistency, but is this needed?
		c.queue(func() {
			if !Compatible(bodyExpr.Type, ov.Return) &&
				bodyExpr.Type.Kind() != InvalidType && bodyExpr.mode&todoExpr == 0 {
				err := returnTypeMismatch(
					bodyExpr.Type, ov.Return,
					stmt.Expression.GetRange(), retRange,
				)
				c.fileError(err, ov.file)
			}
		}, true)
	}
}

// fn could be nil if the overload is an initializer
func (c *Checker) checkFuncBody(stmt *ast.FunctionDeclaration, ov *Overload,
	fn *Function, retRange ranges.Range, fctx *Context,
) {
	sctx := newStmtContext(ov.InnerContext, ov.file, allowReturn)
	sctx.returnHint = ov.Return
	c.checkBlock(stmt.Body.Body, sctx)

	// Ensure return statements are present. They aren't needed if:
	// - The return type is Nothing
	// - The function declares named returns
	// - The function crashouts or has a TODO, or
	// - The function is an initializer (TODO: warn about a missing return
	// if 'self' isn't mutated)
	if len(*sctx.returns) == 0 && ov.Return.Kind() != NothingType &&
		len(ov.NamedReturns) == 0 && sctx.flags&unreachableStmt == 0 &&
		ov.info.funcKind != initFunc {
		err := klarerrs.Position(klarerrs.ErrMissingReturn, stmt.Body.Range.End)
		err.Label = "No 'return' statements in the body"
		err.Name = ov.Return.String()
		err.AddHighlight(
			"This function is supposed to return "+quote(ov.Return.String()),
			retRange,
		)
		c.fileError(err, ov.file)
		return
	}

	// Check that all returned values are compatible with the expected type
	for _, ret := range *sctx.returns {
		if !Compatible(ret.expr.Type, ov.Return) && ret.expr.Type.Kind() != InvalidType {
			err := returnTypeMismatch(ret.expr.Type, ov.Return, ret.pos, retRange)
			c.fileError(err, ov.file)
		}
	}
}

func returnTypeMismatch(got, exp Type, gotRange, expRange ranges.Range) *klarerrs.Error {
	err := typeMismatch(exp, got, gotRange)
	err.Label = "The returned value has type " + quote(got.String())
	err.AddHighlight(
		"The function is supposed to return "+quote(exp.String()),
		expRange,
	)
	return err
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
	writeVariadic := func(p *Variable) bool {
		if p.Object != nil && p.Object.flags&VariadicParam != 0 {
			b.WriteString("...")
			b.WriteString(p.Type.(*List).Elem.String())
			return true
		}
		return false
	}

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
		if writeVariadic(param) {
			continue
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
		if writeVariadic(param.Variable) {
			continue
		}
		b.WriteString(param.Type.String())
	}
	b.WriteByte(')')
	if o.Return != nil && o.Return.Kind() != NothingType && o.Return.Kind() != InvalidType {
		b.WriteString(" -> ")
		b.WriteString(o.Return.String())
	}
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

// checkOverloadAmbiguity checks the given overloads for duplicates and ambiguous
// options (as described in the [Function Overloads] section in the Klar
// Type System). If there are errors, the first of each conflicting overload
// pair is retained in the result.
//
// [Function Overloads]:
func (c *Checker) checkOverloadAmbiguity(overloads []*Overload) []*Overload {
	if len(overloads) == 1 {
		return overloads // Nothing to check
	}
	// Sort by arity so we can make pairs using i+1 and i-1
	byArity := slices.Clone(overloads)
	slices.SortStableFunc(byArity, sortByArity)

	var overloadsWithErrors map[*Overload]struct{}
	addError := func(ov *Overload) {
		if overloadsWithErrors == nil {
			overloadsWithErrors = make(map[*Overload]struct{})
		}
		overloadsWithErrors[ov] = struct{}{}
	}
	// First, check for redeclared overloads
	for i, ov := range byArity {
		var other *Overload
		if i == len(byArity)-1 {
			ov, other = byArity[i-1], ov
		} else {
			other = byArity[i+1]
		}
		// TODO: This has to be looped in a quadratic fashion
		// Otherwise, a setup in this order wouldn't report an error:
		//
		// 	func isNumber(char: String) = char in '0'...'9'
		//  func isNumber(char: Int) = char in '0'...'9'
		//  func isNumber(char: String) = char in '0'...'9'
		if ok := c.checkRedeclaredOverload(ov, other); !ok {
			// The 2nd redeclaration is the one with the error
			err := klarerrs.Range(klarerrs.ErrRedeclaredOverload, other.rang)
			err.Name = ov.String()
			err.Label = "An overload with these same parameters already exists"
			err.AddDetail("It was already declared here", ov.FilePath(), ov.rang)
			c.fileError(err, other.file)
			addError(other)
		}
	}

	if len(overloadsWithErrors) == 0 {
		return overloads // No errors
	}
	// Retain only the overloads without errors
	deduped := make([]*Overload, 0, len(overloads)-len(overloadsWithErrors))
	for _, ov := range overloads {
		if _, ok := overloadsWithErrors[ov]; !ok {
			deduped = append(deduped, ov)
		}
	}
	return deduped
}

func sortByArity(a, b *Overload) int {
	if byMin := a.Arity.MinParams - b.Arity.MinParams; byMin != 0 {
		return byMin
	}
	// -1 is considered greater
	if a.Arity.MaxParams == -1 || b.Arity.MaxParams == -1 {
		return b.Arity.MaxParams - a.Arity.MaxParams
	}
	return a.Arity.MaxParams - b.Arity.MaxParams
}

func (c *Checker) checkRedeclaredOverload(a, b *Overload) (ok bool) {
	typesEqual := func(a, b *Variable) bool { return a.Type == b.Type }
	equal := slices.EqualFunc(a.Params, b.Params, typesEqual) &&
		maps.EqualFunc(a.labelMap, b.labelMap, typesEqual)
	return !equal
}

// TODO: return sorted overloads by ranking
func (c *Checker) resolveOverload(overloads []*Overload,
	params []Type, labelledParams map[string]Type,
) (*Overload, *klarerrs.Error) {
	return overloads[0], nil
}
