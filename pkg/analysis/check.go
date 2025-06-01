package analysis

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type CheckOptions struct {
	OnError         func(err errors.KlarError)
	ContinueOnError bool
}

func CheckProgram(program ast.Program, options CheckOptions) []errors.KlarError {
	c := analysis.NewChecker(program)
	c.OnError = options.OnError
	c.ContinueOnError = options.ContinueOnError
	c.CheckProgram()
	return c.Errors
}
