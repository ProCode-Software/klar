package analysis

import (
	// "github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ast/typedast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/target"
)

type CheckOptions struct {
	OnError  func(err errors.CompileError)
	Target   target.Target
	FilePath string
}

func CheckProgram(program *ast.Program, options CheckOptions) (
	typedProgram *typed.Program, errors []errors.CompileError,
) {
	return nil, nil
	/*
		 	c := analysis.NewChecker(program)
			c.OnError = options.OnError
			c.Target = options.Target
			c.FilePath = options.FilePath
			typed := c.CheckProgram()
			return typed, c.Errors
	*/
}
