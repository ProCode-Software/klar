package module

import (
	"errors"
	"strings"
)

var (
	ErrModuleNotFound     = errors.New("module not found")
	ErrImportCycle        = errors.New("cyclic import")
	ErrImportModuleTooNew = errors.New("module requires a newer Klar version")
)

func StringImportPath(importPath []string) string {
	return strings.Join(importPath, ".")
}
