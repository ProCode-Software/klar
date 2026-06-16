package build

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

var _ analysis.Importer = (*Importer)(nil)

type Importer struct {
	Deps       *Deps
	PkgInfo    *module.PackageInfo
	Targets    []target.Target
	ImportPath imports.ImportPath
	importErrs map[string]error
}

func NewImporter(
	i *Input, importPath imports.ImportPath, deps *Deps,
) *Importer {
	return &Importer{
		ImportPath: importPath,
		Deps:       deps,
		PkgInfo:    i.PkgInfo,
		Targets:    i.Targets,
	}
}

func (i *Importer) Import(p imports.ImportPath, ctx analysis.ImportContext) (
	*analysis.Module, error,
) {
	mod, ok := i.Deps.TryGet(p.String())
	switch {
	case !ok:
		// TODO: offer name hints
		return nil, klarerrs.ImportError(klarerrs.ErrModuleNotFound, p, nil)
	case !i.ImportPath.CanImport(p):
		// Attempt to import a private/internal module
		return nil, klarerrs.ImportError(klarerrs.ErrPrivateImport, p, nil)
	case mod.Failed:
		// The module being imported failed to compile. This error is only
		// shown if importing a dependency (not a project module). For project
		// modules, modules with failed deps are skipped.
		return nil, klarerrs.ImportError(klarerrs.ErrModuleCompileError, p, nil)
	case mod.Checked == nil:
		// Possible toposort bug. This shouldn't happen
		panic(fmt.Sprintf(
			"typed AST of module being imported (%#q) is nil (possible toposort bug)", mod.Path,
		))
	case i.importErrs != nil && i.importErrs[p.String()] != nil:
		// It was passed to the Importer to return an error when the module with
		// this path was imported.
		err := i.importErrs[p.String()]
		if _, ok := err.(*klarerrs.Error); !ok {
			err = klarerrs.ImportError(klarerrs.ErrImporterError, p, err)
		}
		return nil, err
	}
	return mod.Checked, nil
}

/* Use these errors:
klarerrs.ImportError(klarerrs.ErrImportPathConflict, p, dir, nil)
klarerrs.ImportError(klarerrs.ErrModuleCompileError, p, modulePath, err)
klarerrs.ImportError(klarerrs.ErrImporterError, p, "", err)
klarerrs.ImportError(klarerrs.ErrPrivateImport, p, "", nil)
klarerrs.ImportError(klarerrs.ErrModuleNotFound, p, modulePath, err)
*/
