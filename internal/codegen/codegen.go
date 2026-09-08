package codegen

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

type Flags uint8

const (
	// Produce JavaScript suitable for library use
	LibraryMode = 1 << iota
	TypeScript  // Produce type annotations
)

// A Generator converts a Klar module to JavaScript IR.
type Generator struct {
	*analysis.Module
	Files []*File
	Flags Flags
	// TODO: Maybe store a map of TypeScript types for symbols
	// instead of storing types in IR nodes
}

type File struct {
	Name string       // Base name of the file
	AST  *ast.Program // Klar file
	IR   *jsir.Module // JavaScript file
}

func NewGenerator(files map[string]*ast.Program, typedMod *analysis.Module) *Generator {
	codegenFiles := make([]*File, 0, len(files))
	for path, prog := range files {
		codegenFiles = append(codegenFiles, &File{
			Name: path,
			AST:  prog,
			IR: &jsir.Module{
				Decls: make(map[string]jsir.Statement),
			},
		})
	}
	return &Generator{
		Files:  codegenFiles,
		Module: typedMod,
	}
}

func (g *Generator) Run() {
	// Convert everything in each file to JS IR, except top-level methods and initializers
	var wg sync.WaitGroup
	for _, file := range g.Files {
		wg.Go(func() { g.generateFile(file) })
	}
	wg.Wait()

	// Convert methods/initializers and add the to class bodies
}

// Top-level methods and initializers in the file are skipped
func (g *Generator) generateFile(f *File) {
	// TODO: Don't generate dead code
	f.IR.Statements = make([]jsir.Statement, 0, len(f.AST.Body))
	var fid analysis.FileID
	// TODO: loop over g.Info.InitOrder when InitOrder is populated in the type checker
	for _, obj := range g.Module.Context.SortedDecls() {
		if fid == 0 && obj.FileName() == f.Name {
			fid = obj.File // Set the initial file ID for this file
		} else if fid == 0 || obj.File != fid {
			continue
		}
		node := obj.Node().(ast.Statement)
		ctx := obj.Context
		f.IR.Statements = append(f.IR.Statements, g.convertStatement(node, ctx))
	}

	// TODO: Convert comments to offsets of IR nodes
	// f.IR.Comments = make([]jsir.Comment, 0, len(f.AST.Comments))
}
