package codegen

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

type Flags uint8

const (
	// Produce JavaScript suitable for library use
	LibraryMode = 1 << iota
)

// A Generator converts a Klar module to JavaScript IR.
type Generator struct {
	TypedMod *analysis.Module
	Files    []*File
	Flags    Flags

	// Methods and initializers for top-level objects
	methods map[string][]*analysis.Object
}

type File struct {
	Path         string
	*ast.Program              // Klar file
	IR           *jsir.Module // JavaScript file
}

func NewGenerator(files map[string]*ast.Program, typedMod *analysis.Module) *Generator {
	codegenFiles := make([]*File, 0, len(files))
	for path, prog := range files {
		codegenFiles = append(codegenFiles, &File{
			Path:    path,
			Program: prog,
			IR: &jsir.Module{
				Decls: make(map[string]jsir.Statement),
			},
		})
	}
	return &Generator{
		Files:    codegenFiles,
		TypedMod: typedMod,
	}
}

func (g *Generator) Run() {
}
