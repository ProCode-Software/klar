package analysis

import (
	"fmt"
	"slices"

	"github.com/ProCode-Software/klar/internal/module/imports"
)

// Contains the method definitions for builtin types and functions.
var builtinModule *Module

var BuiltInContext = &Context{File: -2}

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

// Builtin functions
var builtinFuncs = []string{"print", "crashout", "clone", "TODO"}

type List struct{ Elem Type }

func (l *List) Kind() Kind     { return KindList }
func (l *List) String() string { return "[" + TypeToString(l.Elem) + "]" }

func (c *Checker) loadInternalModules() {
	var (
		builtinImportPath    = imports.ImportPath{"klar", "_builtin"}
		attributesImportPath = imports.ImportPath{"klar", "_builtin", "attributes"}
	)
	// TODO: Remove when Importer is implemented. An importer will be required to typecheck.
	if c.Options.Importer == nil {
		return
	}
	// The internal module is currently being typechecked
	isBootstrap := c.module.Flags.Has(BootstrapModule)
	currImpPath := c.module.ImportPath

	// Prevent cycles while bootstrapping the internal modules.
	// We want to make primitive types accessible to the attributes module, and
	// vice versa. The primitives module already declares its own objects.
	if !isBootstrap || slices.Equal(currImpPath, attributesImportPath) {
		builtinModule = c.importInternally(builtinImportPath)
		declareBuiltinTypes()
		declareBuiltinFunctions()
	}
	if !isBootstrap || slices.Equal(currImpPath, builtinImportPath) {
		attributesModule = c.importInternally(attributesImportPath)
	}
}

func (c *Checker) importInternally(path imports.ImportPath) *Module {
	mod, err := c.Options.Importer.Import(path, &importCtx{
		target:   c.Options.Target,
		internal: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to perform internal import for %s: %v", path, err))
	}
	return mod
}

func declareBuiltinTypes() {
	for _, p := range primitives {
		BuiltInContext.Declare(&Object{
			name:    p.name,
			context: BuiltInContext,
			typ:     p.typ,
			file:    -2,
		})
	}
}

func declareBuiltinFunctions() {
	for _, name := range builtinFuncs {
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
	crashout.Return = &Unreachable{Type: crashout.Return}
	todo.Return = &Unreachable{Type: todo.Return}
}
