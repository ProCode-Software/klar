package module

import (
	"errors"
	"strings"

	"github.com/ProCode-Software/klar/internal/target"
)

var (
	ErrModuleNotFound = errors.New("module not found")
	ErrImportCycle = errors.New("cyclic import")
	ErrImportModuleTooNew = errors.New("module requires a newer Klar version")
)

// TODO: use analysis.Module & fix cycle

// Importer imports Klar modules.
type Importer interface {
	Import(importPath []string, target target.Target) (any, error)
}

func StringImportPath(importPath []string) string {
	return strings.Join(importPath, ".")
}