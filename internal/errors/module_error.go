package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	_ ErrorCode = ModuleErrorPrefix + iota

	ErrModuleNotFound   // Module not found
	ErrModuleCycle      // Modules depend on each other
	ErrModuleKlarTooNew // Module being imported requires a newer Klar version
	ErrImporterError    // Importer returned a miscellaneous error
	ErrImporterNotFound // Importer not set up
)

type ModuleError struct {
	File       string
	Code       ErrorCode
	Range      ranges.Range
	ImportPath string
	Params     ErrorParams
	Label      string
	Details    []Detail
	Hints      []Hint
	Highlights []Highlight
}

func (err *ModuleError) Error() string {
	return "ModuleError: " + err.error()
}

func (err *ModuleError) error() string {
	switch err.Code {
	default:
		panic("ModuleError: no error message for code " + err.Code.String())
	case ErrModuleNotFound:
		return "Can't find a module named '" + err.ImportPath + "'"
	case ErrImporterError:
		impErr := err.Params["error"].(error)
		return fmt.Sprintf("Failed to import %s: %v", Quote(err.ImportPath), impErr)
	}
}
