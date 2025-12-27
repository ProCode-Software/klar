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
	FileIDs     map[FileID]string
	ImportPath  imports.ImportPath
	Imports     []*Module
	Target      *target.Target   // Target the module was compiled for
	KlarVersion *version.Version // Minimum required Klar version
	Context     *Context         // Root non-builtin context
	Flags       Flag
}

// NewModule returns a new Module. The module is not complete.
func NewModule(
	name, path string,
	importPath imports.ImportPath,
	programs map[string]*ast.Program,
	klarVersion *version.Version,
	target *target.Target,
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

func (m *Module) ImportPathString() string {
	return m.ImportPath.String()
}

func (m *Module) FullFilePath(basename string) string {
	if m.Flags.Has(SingleFileModule) {
		return m.Path
	} else {
		return filepath.Join(m.Path, basename)
	}
}

func (m *Module) FilePathFromID(id FileID) string {
	return m.FullFilePath(m.FileIDs[id])
}

func (m *Module) String() string {
	return fmt.Sprintf("module %s (%s)", m.ImportPathString(), m.Path)
}
