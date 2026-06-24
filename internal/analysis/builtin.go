package analysis

import (
	"fmt"
	"slices"
	"strings"

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
var primitives = []struct {
	name string
	typ  Kind
}{
	{"Int", IntType},
	{"String", StringType},
	{"Bool", BoolType},
	{"Float", FloatType},
	{"Any", AnyType},
	{"Nothing", NothingType},
}

// Composite types
var compositeTypes = []struct {
	declaredName string // Name as declared in the builtin module
	kind         Kind
	asKind       func(*Context) Type // The type that actually has the kind
}{
	{"List", KindList, func(ctx *Context) Type {
		return &List{ctx.Lookup("T")}
	}},
	{"Map", KindMap, func(ctx *Context) Type {
		return &Map{ctx.Lookup("K"), ctx.Lookup("V")}
	}},
	{"Result", KindResult, func(ctx *Context) Type {
		return &Result{ctx.Lookup("T"), ErrorType}
	}},
	{"Task", KindTask, func(ctx *Context) Type {
		return &Task{ctx.Lookup("T")}
	}},
	{"Optional", KindOptional, func(ctx *Context) Type {
		return &Optional{ctx.Lookup("T")}
	}},
	{"Error", ErrorType, func(*Context) Type { return ErrorType }},
}

// Builtin functions
var BuiltinFuncs = []string{"print", "crashout", "clone", "zip", "TODO"}

type Tuple []Type

func (t Tuple) Kind() Kind { return KindTuple }
func (t Tuple) String() string {
	if len(t) == 0 {
		return "()"
	}
	var b strings.Builder
	b.WriteByte('(')
	for i, elem := range t {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(elem.String())
	}
	b.WriteByte(')')
	return b.String()
}

type List struct{ Elem Type }

func (l *List) Kind() Kind     { return KindList }
func (l *List) String() string { return "[" + l.Elem.String() + "]" }

type Map struct{ Key, Value Type }

func (*Map) Kind() Kind { return KindMap }
func (m *Map) String() string {
	return fmt.Sprintf("#{%s: %s}", m.Key.String(), m.Value.String())
}

type Optional struct{ Elem Type }

func (*Optional) Kind() Kind       { return KindOptional }
func (o *Optional) String() string { return o.Elem.String() + "?" }

// TODO: Should Optional have an Underlying() type?

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

// Loading
// ==========

func (c *Checker) loadInternalModules() {
	if builtinModule != nil && attributesModule != nil {
		attributesAllowed = true
		return // Already loaded
	}
	var (
		builtinImportPath    = imports.ImportPath{"klar", "_builtin"}
		attributesImportPath = imports.ImportPath{"klar", "_builtin", "attributes"}
		currImpPath          = c.module.ImportPath
		// True if the internal module is currently being typechecked
		isBootstrap = c.module.Flags.Has(BootstrapModule)
	)
	// As a temporary limitation, the builtin module can't reference attributes.
	// The attributes module needs the builtin types.
	attributesAllowed = !isBootstrap
	if isBootstrap {
		if slices.Equal(currImpPath, builtinImportPath) {
			builtinModule = c.module
		} else if slices.Equal(currImpPath, attributesImportPath) {
			attributesModule = c.module
		}
	}
	if !isBootstrap || slices.Equal(currImpPath, attributesImportPath) {
		declareBuiltinTypes()
		declareBuiltinFunctions()
	}
}

func declareBuiltinTypes() {
	for _, p := range primitives {
		BuiltInContext.Declare(&Object{
			name:    p.name,
			context: BuiltInContext,
			typ:     &TypeName{Type: p.typ},
			file:    -2,
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
	crashout := BuiltInContext.Lookup("crashout").typ.(*Function)
	todo := BuiltInContext.Lookup("TODO").typ.(*Function)
	crashout.Return = &NoReturn{Type: crashout.Return}
	todo.Return = &NoReturn{Type: todo.Return}
}
