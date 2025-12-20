package analysis

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module"
)

type importQueueEntry struct {
	stmt       *ast.ImportStatement
	importPath string
	fileName   string // File that imported it
	ctx        *Context
}

func (c *Checker) importModule(
	importPath []string, importPathString string,
	mods chan importKey, errs chan importErrorKey,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	if c.Options.Importer == nil {
		// Importer not set up
		errs <- importErrorKey{importPathString, &errors.ModuleError{}}
		return
	}
	mod, err := c.Options.Importer.Import(importPath, *c.Options.Target)
	if err != nil {
		errs <- importErrorKey{importPathString, err}
		return
	}
	// TODO: remove when Importer is implemented
	mod = &Module{Name: "fakeModule"}
	mods <- importKey{importPathString, mod.(*Module)}
}

type importKey struct {
	importPath string
	module     *Module
}

type importErrorKey struct {
	importPath string
	err        error
}

// performFileImports creates a new [Context] for each file
// and performs imports.
func (c *Checker) performFileImports(fileContexts map[string]*Context) {
	// Create contexts for each file and collect all imports
	queue := make([]*importQueueEntry, 0, len(c.Programs))
	cache := make(map[string]*Module)
	mods := make(chan importKey)
	errs := make(chan importErrorKey)
	var wg sync.WaitGroup

	for fileName, prog := range c.Programs {
		ctx := fileContexts[fileName]
		// Perform imports for the file
		for i, stmt := range prog.Body {
			imp, ok := stmt.(*ast.ImportStatement)
			if !ok {
				ctx.setAttribute(firstStmtIndex, i)
				break
			}
			impPath := module.StringImportPath(imp.Module)
			queue = append(queue, &importQueueEntry{
				stmt:       imp,
				importPath: impPath,
				fileName:   fileName,
				ctx:        ctx,
			})
			// Do an import if it's not already in cache
			if _, ok := cache[impPath]; !ok {
				wg.Add(1)
				go c.importModule(imp.Module, impPath, mods, errs, &wg)
			}
		}
	}

	go func() {
		wg.Wait()
		close(mods)
		close(errs)
	}()

	// Errors are reported when the import is attempted to be applied per file
	importsWithErrors := make(map[string]error)
	for err := range errs {
		importsWithErrors[err.importPath] = err.err
	}
	// Store imported modules in cache
	for modKey := range mods {
		cache[modKey.importPath] = modKey.module
	}
	// Apply the imports
	for _, item := range queue {
		if err, ok := importsWithErrors[item.importPath]; ok {
			// Report module error for the file
			c.reportImportError(item.fileName, item.importPath, err)
			continue
		}
		c.applyImportedModule(cache[item.importPath], item)
	}
}

func (c *Checker) applyImportedModule(mod *Module, queueEntry *importQueueEntry,
) {
	stmt := queueEntry.stmt
	if stmt.Wildcard {
		// We have to import the submodules
	}
	ns := mod.ImportPath[len(mod.ImportPath)-1]
	if stmt.Alias.IsDiscard() {
		return // Don't do anything
	} else if !stmt.Alias.IsZero() {
		ns = stmt.Alias.Name
	}
	_ = ns
	_ = stmt.UnqualifiedImports
}

func (c *Checker) reportImportError(fileName, importPath string, err error) {
}
