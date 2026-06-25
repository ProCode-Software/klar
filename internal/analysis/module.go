package analysis

import (
	"fmt"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

// A Module describes a Klar module.
type Module struct {
	Name, Path  string // Base name, dir/file path
	Programs    map[string]*ast.Program
	fileID      map[FileID]string   // File ID to key in Programs
	fileContext map[FileID]*Context // File ID to fctx
	ImportPath  imports.ImportPath
	Imports     []*Module
	Target      target.Target    // Target the module was compiled for
	KlarVersion *version.Version // Minimum required Klar version
	Context     *Context         // Root non-builtin context
	Flags       Flag
	TopLevel    []ast.Statement
}

// NewModule returns a new Module. The module is not complete.
func NewModule(
	name, path string,
	importPath imports.ImportPath,
	programs map[string]*ast.Program,
	klarVersion *version.Version,
	target target.Target,
) *Module {
	ctx := NewContext(BuiltInContext, 0)
	return &Module{
		Name:        name,
		Path:        path,
		ImportPath:  importPath,
		Programs:    programs,
		Target:      target,
		KlarVersion: klarVersion,
		Context:     ctx,
	}
}

// ImportPathString returns m's import path as a dot-separated path.
func (m *Module) ImportPathString() string {
	return m.ImportPath.String()
}

// JoinFilePath joins the module's directory location with the given basename.
// If m is a single-file module, it returns the file's path. JoinFilePath does not
// validate that basename is a valid file in the module.
func (m *Module) JoinFilePath(basename string) string {
	if m.Flags.Has(SingleFileModule) {
		return m.Path
	} else {
		return filepath.Join(m.Path, basename)
	}
}

// ResolveFile returns the base name of the file represented by id.
// It panics if the file is not found.
func (m *Module) ResolveFile(id FileID) string {
	if id == 0 {
		return "top-level"
	}
	path, ok := m.fileID[id]
	if !ok {
		panic(fmt.Sprintf("file with id %d not found", id))
	}
	return path
}

// ResolveFilePath returns the full path of the file represented by id,
// or an empty string if not found.
func (m *Module) ResolveFilePath(id FileID) string {
	if file := m.fileID[id]; file != "" {
		return m.JoinFilePath(file)
	}
	return ""
}

func (m *Module) String() string {
	return fmt.Sprintf("module %s (%s)", m.ImportPathString(), m.Path)
}

func (m *Module) HasExports() bool {
	for _, obj := range m.Context.Declarations {
		if obj.public {
			return true
		}
	}
	return false
}
