package analysis

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

type CheckOptions struct {
	File        string
	Target      target.Target
	KlarVersion *version.Version
	Path        string
	*analysis.Options
}

func CheckProgram(prog *ast.Program, opts CheckOptions) []*klarerrs.Error {
	mod := analysis.NewModule(
		opts.File, opts.Path, nil,
		map[string]*ast.Program{opts.File: prog},
		opts.KlarVersion, opts.Target,
	)
	c := analysis.NewChecker(mod, opts.Options)
	c.Check()
	return c.Errors
}
