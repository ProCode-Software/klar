package klarerrs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

const (
	_ Code = ModuleErrorPrefix + iota

	ErrModuleNotFound          // Module not found
	ErrImportCycle             // Modules depend on each other
	ErrSelfImport              // Module imports itself
	ErrModuleKlarTooNew        // Module being imported requires a newer Klar version
	ErrImporterError           // Importer returned a miscellaneous error
	ErrImporterNotFound        // Importer not set up
	ErrModuleCompileError      // Module failed to compile
	ErrPrivateImport           // Can't import a private module
	ErrSingleFileImport        // Can't import from a single-file module
	ErrUnsupportedImportTarget // Current target doesn't support importing a module
	ErrImportPathAliased       // You must use the aliased import path when provided
	ErrImportEmpty             // Module being imported has no files or exports
)

func (e *Error) handleModuleError() string {
	i := e.ModuleErrorInfo()
	path := Quote(i.ImportPath)
	switch e.Code {
	default:
		e.noMessage()
		return ""
	case ErrModuleNotFound:
		base, _, _ := strings.Cut(i.ImportPath, ".")
		if slices.Contains(imports.StdlibImports, base) {
			return "Can't find a module named " + path + " in the standard library"
		}
		return "Can't find a module named " + Quote(base)
	case ErrImporterError:
		return fmt.Sprintf("Failed to import %s: %v", path, i.Err)
	case ErrImporterNotFound:
		return "Imports aren't allowed from this file"
	case ErrSingleFileImport:
		return "Single-file modules can only import from the standard library"
	case ErrPrivateImport:
		return path + " is private to another package and can't be imported"
	case ErrModuleCompileError:
		e.Hint("If you aren't the author of the module, this isn't your fault. Consider reporting an issue to the author of the library.")
		return "Module " + path + " failed to compile"
	case ErrModuleKlarTooNew:
		expKlar := e.StringParam("expKlar")
		currKlar := e.StringParam("currKlar")
		return fmt.Sprintf(
			"Module %s requires Klar %s or later, but the current module targets %s",
			path, expKlar, currKlar,
		)
	case ErrUnsupportedImportTarget:
		return "Can't import " + path + " because it doesn't support the " +
			e.GetParam("currTarget").(target.Target).String() + " target"
	case ErrSelfImport:
		// Module my.mod directly imports my.mod
		return "Module " + path + " can't import itself!"
	}
}
