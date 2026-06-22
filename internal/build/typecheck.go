package build

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

func (pkc *PackageCompiler) TypeCheckModule(
	m *Module, importPathStr string,
) []*klarerrs.Error {
	importPath := imports.NewImportPath(importPathStr)

	// Set a default target for the input if there isn't one already
	if len(pkc.Input.Targets) == 0 {
		pkc.Input.Targets = []target.Target{0}
	}
	// Type checker options from klar.build
	opts := pkc.getCheckerOptions(m)
	importer := NewImporter(pkc.Input, importPath, pkc.Deps)
	importer.importErrs = pkc.importErrs
	opts.Importer = importer

	// Initialized the typed module
	mod := analysis.NewModule(
		m.Name(), m.Path, importPath,
		m.Programs,
		opts.KlarVersion, opts.Target,
	)
	if m.SingleFile {
		mod.Flags |= analysis.SingleFileModule
	}
	// Apply bootstrap flag for stdlib modules being bootstrapped
	// (klar._builtin and klar._builtin.attributes)
	if isBootstrapping && len(importPath) > 1 &&
		importPath[0] == "klar" && importPath[1] == "_builtin" {
		mod.Flags |= analysis.BootstrapModule
	}

	ch := typeCheckerPool.Get(mod, opts)
	defer typeCheckerPool.Put(ch)
	ch.Check()
	m.Checked = ch.CheckedModule()
	return ch.Errors
}

// getCheckerOptions returns an [analysis.Options] object for a module. It
// contains options from klar.build and the parent package's manifest.
func (pkc *PackageCompiler) getCheckerOptions(mod *Module) *analysis.Options {
	var checkerOptions *klarbuild.CheckerOptions
	if pkc.KlarBuild != nil {
		checkerOptions = pkc.KlarBuild.Checker
	}
	opts := &analysis.Options{
		CheckerOptions:       checkerOptions,
		Target:               pkc.Targets[0], // TODO
		IsTest:               pkc.Mode == ModeTest,
		MaxErrors:            MaxErrors,
		KlarVersion:          nil, // TODO: [version.Specifier].Min()
		EnforceTargetSupport: pkc.EnforceTargetSupport,
	}
	return opts
}

// Pool of [analysis.Checker] objects (similar to [parsePool])
// =========

var typeCheckerPool = newCheckerPool()

type checkerPool struct{ sync.Pool }

func newCheckerPool() *checkerPool {
	return &checkerPool{sync.Pool{
		New: func() any { return new(analysis.Checker) },
	}}
}

// Get returns an [analysis.Checker] from the pool, initializing it with mod and opts.
func (p *checkerPool) Get(
	mod *analysis.Module, opts *analysis.Options,
) *analysis.Checker {
	ch := p.Pool.Get().(*analysis.Checker)
	ch.Init(mod, opts)
	return ch
}

// Put resets ch and puts it back into the pool.
func (p *checkerPool) Put(ch *analysis.Checker) {
	ch.ResetAll()
	p.Pool.Put(ch)
}
