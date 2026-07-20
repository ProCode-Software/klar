package analysis

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
)

// This file declares the builtin types and functions from the Klar language.
// It loads the builtin module (`klar._builtin`, bootstrapped)
// and the attributes module `klar._builtin.attributes` for use by the checker.
//
// Definitions: "Primitive" types are those that don't depend on other types,
// while "builtins" refer to both types and functions that don't need to
// be imported from another stdlib module. Primitive types, and types that
// depend on other types (lists, maps, functions, etc.), are included.
// =======

// Contains the method definitions for builtin types and functions.
// Bootstrapped and typechecked Klar module.
var builtinModule *Module

var BuiltInContext = &Context{File: -1}

// Types that always stay the same and don't depend on other types.
//
// Builtin types that are excluded from this list:
// - List, Map, Result, Error (TODO)
var primitives = map[string]Kind{
	"Int":     IntType,
	"String":  StringType,
	"Bool":    BoolType,
	"Float":   FloatType,
	"Any":     AnyType,
	"Nothing": NothingType,
}

// Composite types
// Keys are the names as declared in the builtin module
var compositeTypes = map[string]struct {
	kind   Kind
	asKind func(*Context) Type // The type that actually has the kind
}{
	"List": {KindList, func(ctx *Context) Type { return &List{ctx.Lookup("T").Type} }},
	"Map": {KindMap, func(ctx *Context) Type {
		return &Map{ctx.Lookup("K").Type, ctx.Lookup("V").Type}
	}},
	"Result": {KindResult, func(ctx *Context) Type {
		return &Result{ctx.Lookup("T").Type, ErrorType} // TODO: Change to ctx.Lookup("E")
	}},
	"Task": {KindTask, func(ctx *Context) Type { return &Task{ctx.Lookup("T").Type} }},
	"Optional": {KindOptional, func(ctx *Context) Type {
		return &Optional{ctx.Lookup("T").Type}
	}},
	"Error": {ErrorType, func(*Context) Type { return ErrorType }},
}

// Builtin functions
var BuiltinFuncs = []string{"print", "crashout", "clone", "zip", "TODO"}

type Tuple struct{ Items []Type }

func (t *Tuple) Kind() Kind { return KindTuple }
func (t *Tuple) String() string {
	if len(t.Items) == 0 {
		return "()"
	}
	var b strings.Builder
	b.WriteByte('(')
	for i, elem := range t.Items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(elem.String())
	}
	b.WriteByte(')')
	return b.String()
}
func (tup *Tuple) Len() int { return len(tup.Items) }
func (tup *Tuple) IndexComputed(index Type, t *Expr) *klarerrs.Error {
	if index.Kind() != IntType {
		return indexTypeMismatchError(
			klarerrs.ErrNonNumericIndex,
			KindTuple, index, "Can't index a tuple using type "+index.String(),
		)
	}
	// TODO: Constant analysis to get the actual item at the index. For now,
	// indexing a tuple returns a union of the tuple's elements.
	if len(tup.Items) == 0 {
		t.Type = InvalidType
	} else {
		t.Type = &Union{Types: tup.Items}
	}
	return nil
}

type List struct{ Elem Type }

func (l *List) Kind() Kind     { return KindList }
func (l *List) String() string { return "[" + l.Elem.String() + "]" }

func (l *List) Index(f string, t *Expr) *klarerrs.Error {
	err := indexBuiltin("List", f, t)
	// Add a hint to use `list += [item]` instead of `list.append(item)`
	// TODO: Line diff
	if err != nil && f == "append" {
		err.Hint("Use += to append to a list.")
	}
	if err == nil {
		// TODO: This may actually mutate the original signature
		t.Type = Substitute(t.Type, map[Type]Type{lookupBootstrap("T"): l.Elem})
	}
	return err
}

func (l *List) IndexComputed(i Type, t *Expr) *klarerrs.Error {
	if i.Kind() != IntType {
		return indexTypeMismatchError(
			klarerrs.ErrNonNumericIndex,
			KindList, i, "Can't index a list using type "+i.String(),
		)
	}
	t.Type = l.Elem
	// TODO: constant analysis (negative index, out of range index)
	return nil
}

type Map struct{ Key, Value Type }

func (*Map) Kind() Kind { return KindMap }
func (m *Map) String() string {
	return fmt.Sprintf("#{%s: %s}", m.Key.String(), m.Value.String())
}

func (m *Map) IndexComputed(i Type, t *Expr) *klarerrs.Error {
	if Compatible(i, m.Key) {
		t.Type = &Optional{m.Value}
		return nil
	}
	if Underlying(i) == Untyped(KindOptional) {
		// Nil literal key
		return indexError(klarerrs.ErrNilMapIndex, i, "The result is always 'none'")
	}
	err := indexTypeMismatchError(
		klarerrs.ErrInvalidMapIndex, m.Key, i, "This index has type "+quote(i.String()),
	)
	err.Name = m.String()
	return err
}

func (m *Map) Index(i string, t *Expr) *klarerrs.Error {
	builtinErr := indexBuiltin("Map", i, t)
	if builtinErr == nil {
		t.Type = Substitute(t.Type, map[Type]Type{
			lookupBootstrap("K"): m.Key,
			lookupBootstrap("V"): m.Value,
		})
		return nil
	}
	// For maps, `m.key` is the same as `m['key']`. Builtin fields have
	// precedence over map keys, so this only runs for unknown fields.
	//
	// TODO: Should we disallow this from the language? A user can misspell
	// a builtin field (such as `lenth`) and if the Map's key type is String,
	// it will silently succeed. Or warn if the field is similar to a builtin?
	if m.Key.Kind() == StringType {
		t.Type = &Optional{m.Value}
		return nil
	}
	// If it isn't a String key, always look for a field on the Map
	// builtin, so return the error from that
	return builtinErr
}

type Optional struct{ Elem Type }

func NewOptional(elem Type) *Optional {
	if elem.Kind() == KindOptional {
		// Note: Aliases aren't preserved
		return As[*Optional](elem)
	}
	return &Optional{elem}
}

func (*Optional) Kind() Kind       { return KindOptional }
func (o *Optional) String() string { return o.Elem.String() + "?" }

type Result struct{ Success, Error Type }

var ResultNothing = &Result{Success: NothingType, Error: ErrorType}

func (*Result) Kind() Kind { return KindResult }
func (r *Result) String() string {
	// Shorthands
	switch {
	case r.Success == NothingType && r.Error == ErrorType:
		return "Result"
	case r.Error == ErrorType:
		return "Result<" + r.Success.String() + ">"
	}
	return fmt.Sprintf("Result<%s, %s>", r.Success.String(), r.Error.String())
}

type Task struct{ Result Type }

func (*Task) Kind() Kind { return KindTask }
func (t *Task) String() string {
	return "Task<" + t.Result.String() + ">"
}

func (t *Task) Index(f string, e *Expr) *klarerrs.Error {
	err := indexBuiltin("Task", f, e)
	if err == nil {
		e.Type = Substitute(e.Type, map[Type]Type{lookupBootstrap("T"): t.Result})
	}
	return err
}

// Loading
// ==========

func (c *Checker) loadInternalModules() {
	var (
		builtinImportPath    = imports.ImportPath{"klar", "_builtin"}
		attributesImportPath = imports.ImportPath{"klar", "_builtin", "attributes"}
		currImpPath          = c.module.ImportPath
	)
	switch {
	// BootstrapModule is set if the module containing builtins or
	// attributes is currently being typechecked
	case !c.module.Flags.Has(BootstrapModule) && !builtinsLoaded:
		// Not bootstrapping. These only need to be declared once per
		// compile session.
		declareBuiltinTypes()
		declareBuiltinFunctions()
		builtinsLoaded = true
	case slices.Equal(currImpPath, builtinImportPath):
		builtinModule = c.module
	case slices.Equal(currImpPath, attributesImportPath):
		attributesModule = c.module
	}
}

var builtinsLoaded bool

func declareBuiltinTypes() {
	for name, kind := range primitives {
		BuiltInContext.Declare(&Object{
			Name: name,
			Type: &TypeName{Type: kind},
			File: BuiltInContext.File,
		})
	}
}

func declareBuiltinFunctions() {
	for _, name := range BuiltinFuncs {
		obj := builtinModule.Context.Lookup(name)
		if obj == nil {
			panic(fmt.Sprintf("builtin function %s not found", name))
		}
		BuiltInContext.Declare(obj)
	}
	// TODO: TODO() is assignable to any value.
	// Change the return types for `crashout` and `TODO` so that any statement
	// after a call to them should be deemed unreachable.
	crashout := BuiltInContext.Lookup("crashout").Type.(*Function)
	todo := BuiltInContext.Lookup("TODO").Type.(*Function)
	crashout.Return = &NoReturn{Type: crashout.Return}
	todo.Return = &NoReturn{Type: todo.Return}
}
