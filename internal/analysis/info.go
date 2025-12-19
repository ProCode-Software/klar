package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

// A Module describes a Klar module.
type Module struct {
	Name, Path  string // Base name, dir/file path
	Programs    map[string]*ast.Program
	ImportPath  []string
	Imports     []*Module
	Target      *target.Target   // Target the module was compiled for
	KlarVersion *version.Version // Minimum required Klar version
	Context     *Context         // Root non-builtin context
	Flags       Flag
}

// NewModule returns a new Module. The module is not complete.
func NewModule(
	name, path string,
	importPath []string,
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
