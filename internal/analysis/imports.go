package analysis

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

// Importer resolves import paths to [Module]s.
type Importer interface {
	Import(imports.ImportPath, ImportContext) (*Module, error)
}

type ImportContext interface {
	Target() target.Target
	// The import path of the module that is importing.
	ImportPath() imports.ImportPath
	// The folder path of the module that is importing.
	DirPath() string
	// Whether the module that is importing is a single file.
	SingleFile() bool
}

// importCtx is the implementation of [ImportContext].
type importCtx struct {
	target     target.Target
	importPath imports.ImportPath
	fileDir    string
	singleFile bool
}

func (c *importCtx) Target() target.Target          { return c.target }
func (c *importCtx) ImportPath() imports.ImportPath { return c.importPath }
func (c *importCtx) DirPath() string                { return c.fileDir }
func (c *importCtx) SingleFile() bool               { return c.singleFile }

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
		queue  = make([]*importQueueEntry, 0, len(c.Programs)) //
		mods   = make(map[string]imported)
		wg     sync.WaitGroup
		modsMu sync.Mutex
		impCtx = &importCtx{
			target:     c.module.Target,
			importPath: c.module.ImportPath,
			fileDir:    c.module.Path,
			singleFile: c.module.Flags.Has(SingleFileModule),
		}
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
					mod, err := c.importModule(imp.Module, impCtx)
					modsMu.Lock()
					mods[impPathStr] = imported{mod, err}
					modsMu.Unlock()
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

func (c *Checker) importModule(p imports.ImportPath, impCtx *importCtx) (*Module, error) {
	if c.Options.Importer == nil {
		// Importer not set up
		return nil, &klarerrs.Error{Code: klarerrs.ErrImporterNotFound}
	}
	return c.Options.Importer.Import(p, impCtx)
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
