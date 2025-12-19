package analysis

import (
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

type Options struct {
	// Imports modules. If set to nil, [StdImporter] is used. TODO
	Importer module.Importer
	// The target platform for which the program is being compiled.
	// This is needed for resolving platform-specific implementations
	// and modules.
	Target *target.Target
	// The minimum version of Klar required to compile the program.
	KlarVersion *version.Version
	// If Error != nil, it is called when an error is reported.
	Error func(errors.CompileError)
	// Type checker options from klar.build
	*klarbuild.CheckerOptions
}

type Checker struct {
	Programs    map[string]*ast.Program // Files in the module that is being checked.
	Errors      []errors.CompileError   // Errors reported while type checking.
	Options     *Options                // Options for type checking.
	rootContext *Context                // Context where top-level objects are defined.
	module      *Module
	usedImports map[*Module]struct{}
	importMap   map[string]*Module
}

func NewChecker(
	name, path string, importPath []string,
	files map[string]*ast.Program, opts *Options,
) *Checker {
	if opts == nil {
		opts = &Options{Importer: nil} // TODO: use default importer
	}
	mod := NewModule(name, path, importPath, files, opts.KlarVersion, opts.Target)
	return NewCheckerFromModule(mod, opts)
}

func NewCheckerFromModule(mod *Module, opts *Options) *Checker {
	if opts == nil {
		opts = &Options{Importer: nil} // TODO: use default importer
	}
	return &Checker{
		rootContext: mod.Context,
		module:      mod,
		Programs:    mod.Programs,
		Options:     opts,
	}
}

func NewEmptyChecker() *Checker {
	return &Checker{}
}

func (c *Checker) Init(mod *Module, opts *Options) {
	c.rootContext = mod.Context
	c.module = mod
	c.Programs = mod.Programs
	c.Options = opts
}

func (c *Checker) Reset() {
	c.module = nil
	c.Errors = nil
	c.rootContext = nil
	c.usedImports = nil
	c.Programs = nil
	c.Options = nil
}

func (c *Checker) filePath(name string) string {
	if (c.module.Flags & SingleFileModule) != 0 {
		return c.module.Path
	}
	return filepath.Join(c.module.Path, name)
}

func (c *Checker) Check() {
	c.initFileContextsAndImports()
}
