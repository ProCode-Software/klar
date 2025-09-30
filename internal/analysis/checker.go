package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type File struct {
	Path    string
	Program *ast.Program
}

type ModuleChecker struct {
	Contexts map[ContextID]*Context
	Files    []File
	Errors  []errors.KlarError
	
}

type Checker struct {
	Program *ast.Program
	
}
