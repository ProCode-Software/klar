package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module"
)

type importQueueEntry struct {
	importPath []string
	fileName   string // File that imported it
	ctx        *Context
}

func (c *Checker) importModule(importPath []string) (*Module, error) {
	importPathStr := module.StringImportPath(importPath)
	if mod, ok := c.importMap[importPathStr]; ok {
		return mod, nil
	}
	if c.Options.Importer == nil {
		// Importer not set up
		return nil, &errors.ModuleError{}
	}
	mod, err := c.Options.Importer.Import(importPath, *c.Options.Target)
	return mod.(*Module), err
}

// initFileContextsAndImports creates a new [Context] for each file
// and performs imports.
func (c *Checker) initFileContextsAndImports() {
	// Create contexts for each file and collect all imports
	queue := make([]importQueueEntry, 0, len(c.Programs))
	for fileName, prog := range c.Programs {
		_ = fileName
		ctx := NewContext(c.rootContext, 0)
		ctx.Attrs = map[string]any{"file": fileName}
		// Perform imports for the file
		for _, stmt := range prog.Body {
			imp, ok := stmt.(*ast.ImportStatement)
			if !ok {
				break
			}
			queue = append(queue, importQueueEntry{
				importPath: imp.Module,
				fileName:   fileName,
				ctx:        ctx,
			})
		}
	}
	// TODO: import them concurrently
}
