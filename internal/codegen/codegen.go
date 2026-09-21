// Package codegen performs lowering of typed Klar ASTs to JavaScript
// immediate representation (IR).
package codegen

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

type Flags uint8

const (
	LibraryMode = 1 << iota // Produce JavaScript suitable for library use
	TypeScript              // Produce type annotations
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

func NewGenerator(
	files map[string]*ast.Program, sortedFiles []string,
	typedMod *analysis.Module, flags ...Flags,
) *Generator {
	codegenFiles := make([]*File, len(files))
	for i, baseName := range sortedFiles {
		codegenFiles[i] = &File{
			Name: baseName,
			AST:  files[baseName],
			IR:   &jsir.Module{Decls: make(map[string]jsir.Statement)},
		}
	}
	var f Flags
	for _, flag := range flags {
		f |= flag
	}
	return &Generator{
		Files:  codegenFiles,
		Module: typedMod,
		Flags:  f,
	}
}

func (g *Generator) Run() {
	// Convert everything in each file to JS IR, except top-level methods and initializers
	var wg sync.WaitGroup
	for i, file := range g.Files {
		// The file ID is the 1-based sorted index. See [analysis.Checker.initFileContexts]
		wg.Go(func() { g.generateFile(file, analysis.FileID(i+1)) })
	}
	wg.Wait()

	// Convert methods/initializers and add the to class bodies
}

// Top-level methods and initializers in the file are skipped
func (g *Generator) generateFile(f *File, fid analysis.FileID) {
	// TODO: Don't generate dead code
	f.IR.Statements = make([]jsir.Statement, 0, len(f.AST.Body))

	// TODO: Factor the code so it can be used in nested contexts.
	// In those cases, statements must be converted in order, without
	// separating declarations from other statements.

	// Anything not in this map is a top-level statement
	declStmts := make(map[ast.Statement]struct{}, len(g.Context.SortedDecls()))
	// Top-level declarations
	// TODO: loop over g.Info.InitOrder when InitOrder is populated in
	// the type checker
	for _, obj := range g.Module.Context.SortedDecls() {
		declStmts[obj.Node()] = struct{}{}
		if obj.File != fid || !g.canConvert(obj, true) {
			continue
		}
		jsStmt := g.convertObject(obj)
		if obj.Public {
			jsStmt = &jsir.ExportModifierStatement{Declaration: jsStmt}
		}
		f.IR.Statements = append(f.IR.Statements, jsStmt)
	}

	// Top-level statements that aren't declarations
	for _, stmt := range f.AST.Body {
		if _, ok := stmt.(*ast.PublicDeclaration); ok {
			continue // Public declarations are always Objects
		}
		if _, ok := declStmts[stmt]; !ok {
			jsStmt := g.convertStatement(stmt, g.Context)
			f.IR.Statements = append(f.IR.Statements, jsStmt)
		}
	}

	// TODO: Convert comments to offsets of IR nodes
	// f.IR.Comments = make([]jsir.Comment, 0, len(f.AST.Comments))
}

func (g *Generator) canConvert(obj *analysis.Object, skipMethods bool) bool {
	switch node := obj.Node().(type) {
	case *ast.FunctionDeclaration:
		return !skipMethods || node.SelfType == nil
	case *ast.FuncAliasDeclaration:
		return !skipMethods || node.SelfType == nil
	case *ast.InterfaceDeclaration:
		// Interfaces can't be converted to JavaScript
		return (g.Flags & TypeScript) != 0
	case *ast.TypeAliasDeclaration:
		// Only public type aliases that refer to user-declared concrete types
		// can be converted to JS
		if !obj.Public {
			return false
		}
		switch obj.Kind() {
		case analysis.KindStruct, analysis.KindEnum, analysis.KindTag:
			return true
		default:
			return false
		}
	}
	return true
}
