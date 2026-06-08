package build

import (
	"time"

	"github.com/ProCode-Software/klar/internal/klarerrs"
)

type ProjectCompiler struct {
	*Compiler
	Inputs []ProjectInput
}

type Result struct {
	Modules []*Module
	Errors  []*klarerrs.Error
	Elapsed time.Duration
}

func NewProjectCompiler(c *Compiler) *ProjectCompiler {
	return &ProjectCompiler{Compiler: c}
}

func (pc *ProjectCompiler) Compile() (*Result, error) {
	// Compile() may be called multiple times (such as by the LSP)
	pc.ResetState()
	
	// Dependencies are compiled first
	if err := pc.CompileDeps(); err != nil {
		return nil, err
	}
	// TODO: Reset errors?
	// Then, the inputs from the command line
	if err := pc.CompileInputs(); err != nil {
		return nil, err
	}
	
	return &Result{
		Modules: nil,
		Errors:  pc.Errors,
		Elapsed: time.Since(pc.StartTime),
	}, nil
}

func (pc *ProjectCompiler) CompileDeps() error {
	return nil
}

func (pc *ProjectCompiler) CompileInputs() error {
	return nil
}

func (pc *ProjectCompiler) ResetState() {
	pc.Compiler.ResetState()
}
