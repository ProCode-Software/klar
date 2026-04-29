package analysis

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

// Importer resolves import paths to [Module]s.
type Importer interface {
	Import(importPath imports.ImportPath, target target.Target) (*Module, error)
}

type ImportContext interface {
	Target() target.Target
	ImportPath() imports.ImportPath
}

type importQueueEntry struct {
	stmt       *ast.ImportStatement
	importPath string
	fileName   string // File that imported it
	ctx        *Context
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
func (c *Checker) performFileImports(files []string, fileContexts map[string]*Context) {
	// Create contexts for each file and collect all imports
	type imported struct {
		mod *Module
		err error
	}
	var (
		queue = make([]*importQueueEntry, 0, len(c.Programs)) //
		mods  = make(map[string]imported)
		wg    sync.WaitGroup
	)
	for _, fileName := range files {
		ctx := fileContexts[fileName]
		// Perform imports for the file
		for i, stmt := range c.Programs[fileName].Body {
			imp, ok := stmt.(*ast.ImportStatement)
			if !ok {
				ctx.setAttribute(firstStmtIndex, i)
				break
			}
			impPathStr := imports.ImportPath.String(imp.Module)
			queue = append(queue, &importQueueEntry{
				stmt:       imp,
				importPath: impPathStr,
				fileName:   fileName,
				ctx:        ctx,
			})
			// Do an import if it's not already in cache
			if _, ok := mods[impPathStr]; !ok {
				wg.Go(func() {
					mod, err := c.importModule(imp.Module)
					mods[impPathStr] = imported{mod, err} // TODO: concurrent map write error?
				})
			}
		}
	}
	wg.Wait()

	// Apply the imports
	for _, item := range queue {
		m := mods[item.importPath]
		if m.err != nil {
			// Report module error for the file
			c.reportImportError(item.fileName, item.importPath, m.err)
			continue
		}
		c.applyImportedModule(m.mod, item)
	}
}

func (c *Checker) importModule(p imports.ImportPath) (*Module, error) {
	if c.Options.Importer == nil {
		// Importer not set up
		return nil, &errors.ModuleError{}
	}
	return c.Options.Importer.Import(p, c.Options.Target)
}

func (c *Checker) applyImportedModule(mod *Module, queueEntry *importQueueEntry,
) {
	stmt := queueEntry.stmt
	if stmt.Alias.IsDiscard() {
		return // Don't do anything
	}
	if stmt.Wildcard {
		// We have to import the submodules
	}
	ns := mod.ImportPath[len(mod.ImportPath)-1]
	if !stmt.Alias.IsZero() {
		ns = stmt.Alias.Name
	}
	_ = ns
	_ = stmt.UnqualifiedImports
}

func (c *Checker) reportImportError(fileName, importPath string, err error) {
}
