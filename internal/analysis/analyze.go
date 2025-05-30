package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/runtime"
)

type Checker struct {
	Errors   []errors.KlarError
	Program  ast.Program
	Contexts runtime.ContextMap
}

func NewChecker(program ast.Program) *Checker {
	return &Checker{
		Program:  program,
		Contexts: make(runtime.ContextMap),
	}
}

func (c *Checker) Check() {
	for _, dec := range c.Program.Body {
		
	}
}