package analysis

import (
	// "github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ast/typed"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/target"
)

type CheckOptions struct {
	OnError  func(err errors.KlarError)
	Target   target.Double
	FilePath string
}

func CheckProgram(program *ast.Program, options CheckOptions) (
	typedProgram *typed.Program, errors []errors.KlarError,
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
