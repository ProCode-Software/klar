package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type FileID int

type Checker struct {
	Contexts  map[ContextID]*Context
	filePaths map[FileID]string
	Programs  []*ast.Program
	Errors    []errors.CompileError
}
