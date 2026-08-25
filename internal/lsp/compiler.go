package lsp

import (
	"log/slog"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
)

func (s *Server) compileModule(path string) {
	mod := s.modules[path]
	deps := &s.pkgs[mod.PkgPath].deps

	defer s.compiler.ResetState()
	s.compiler.StartTime = time.Now()
	// TODO: Remove the compilation error limit
	var fatalErr error
	if *deps == nil {
		// First time compiling the package. Compile it to get the
		// dependency packages. This only occurs once per *package*.
		// For that reason, I don't think that a pool is needed.
		pc := build.NewProjectCompiler(s.compiler)
		pc.Inputs = []*build.Input{mod.compilerInput}
		var res *build.Result
		res, fatalErr = pc.Compile()
		*deps = *res.AllModules[mod.compilerInput]
	} else {
		// TODO: Should each module get its own permanent PackageCompiler?
		// It could possibly be deleted when the module is closed.
		pkc := compilerPool.Get()
		defer compilerPool.Put(pkc)
		pkc.Compiler, pkc.Input, pkc.Deps = s.compiler, mod.compilerInput, deps
		_, fatalErr = pkc.Compile()
	}
	if fatalErr != nil {
		s.Error("Fatal error while compiling modules", slog.Any("error", fatalErr))
		// TODO: Send error notification to user
	}

	// The module's dependencies have been compiled, so we can apply them.
	// As a result, they don't have to be compiled when they're first opened.
	//
	// The deps map from the compiler includes the input, so this also applies
	// the typechecked module.
	for _, mod := range *deps {
		if _, ok := s.modules[mod.Path]; !ok {
			// [Module.compilerInput] will be set when the module is opened
			s.modules[mod.Path] = &Module{PkgPath: mod.Path}
		}
		s.modules[mod.Path].Module = mod.Checked
	}

	// Report diagnostics
	s.reportDiagnostics(s.compiler.Errors, s.compiler.Warnings)
}

// initCompiler initializes the shared compiler instance.
func (s *Server) initCompiler() {
	c := build.NewCompiler(build.ModeAnalyze, "")
	c.FS = s.fs
	c.Logger = s.Logger.With(slog.String("source", "compiler"))
	c.UseStdParser()
	s.compiler = c
}

// Pool for module compilers
// ======

var compilerPool = _compilerPool{New: func() any {
	return build.NewPackageCompiler(nil, nil)
}}

type _compilerPool struct {
	sync.Pool
}

func (pool *_compilerPool) Get() *build.PackageCompiler {
	return pool.Pool.Get().(*build.PackageCompiler)
}

func (pool *_compilerPool) Put(pkc *build.PackageCompiler) {
	pkc.Reset()
	pool.Pool.Put(pkc)
}
