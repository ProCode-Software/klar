package klarerrs

import (
	"fmt"
)

const (
	_ Code = ModuleErrorPrefix + iota

	ErrModuleNotFound   // Module not found
	ErrModuleCycle      // Modules depend on each other
	ErrModuleKlarTooNew // Module being imported requires a newer Klar version
	ErrImporterError    // Importer returned a miscellaneous error
	ErrImporterNotFound // Importer not set up
)

func (e *Error) handleModuleError() string {
	importPath := e.StringParam("importPath")
	switch e.Code {
	default:
		e.noMessage()
		return ""
	case ErrModuleNotFound:
		return "Can't find a module named " + Quote(importPath)
	case ErrImporterError:
		impErr := e.Params["error"].(error)
		return fmt.Sprintf("Failed to import %s: %v", Quote(importPath), impErr)
	}
}
