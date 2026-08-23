package lsp

import (
	"log/slog"

	"github.com/ProCode-Software/klar/internal/build"
)

func (s *Server) compilePackages() {
	for _, pkg := range s.pkgs {
		if pkg.Compiler == nil {
			pkg.Compiler = s.createLSPCompiler(pkg.Dir)
		}
		_, _ = pkg.Compiler.Compile()
	}
}

func (s *Server) createLSPCompiler(cwd string) *build.ProjectCompiler {
	c := build.NewCompiler(build.ModeAnalyze, cwd)
	c.FS = s.fs
	c.Logger = s.Logger.With(slog.String("source", "compiler"))
	c.UseStdParser()
	pc := build.NewProjectCompiler(c)
	return pc
}
