package module

import (
	"errors"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/module/imports"
)

type ImportContext = analysis.ImportContext

var (
	ErrModuleNotFound     = errors.New("module not found")
	ErrImportCycle        = errors.New("cyclic import")
	ErrImportModuleTooNew = errors.New("module requires a newer Klar version")
)

// BaseImporter is the default importer that resolves imports from the local filesystem.
type BaseImporter struct {
	Manifest *glaspack.Manifest
	PkgDir   string
}

func (i *BaseImporter) Import(p imports.ImportPath, ctx ImportContext) (
	mod *analysis.Module, err error,
) {
	return
}
