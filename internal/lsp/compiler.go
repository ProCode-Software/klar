package lsp

import (
	"log/slog"
	"sync"

	"github.com/ProCode-Software/klar/internal/build"
)

func (s *Server) compileModule(path string) {
	mod := s.modules[path]
	deps := &s.pkgs[mod.PkgPath].deps
	if *deps == nil {
		// First time compiling the package. Compile it to get its deps
		if s.compiler == nil {
			s.initCompiler() // The server uses a shared [*build.Compiler] instance
		}
		pc := build.NewProjectCompiler(s.compiler)
		pc.Inputs = []*build.Input{mod.compilerInput}
	} else {
		pkc := compilerPool.Get()
		defer compilerPool.Put(pkc)
		pkc.Input, pkc.Deps = mod.compilerInput, deps
	}
	/* if err != nil {
		s.Error("Fatal error while compiling modules", slog.Any("error", err))
		// TODO: Send error notification to user
		continue
	} */
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
