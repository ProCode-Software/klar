package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

// Importer resolves import paths to [Module]s.
type Importer interface {
	Import(imports.ImportPath, ImportContext) (*Module, error)
	// TODO: import wildcard that returns []*Module
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
	internal   bool
}

func (c *importCtx) Target() target.Target          { return c.target }
func (c *importCtx) ImportPath() imports.ImportPath { return c.importPath }
func (c *importCtx) DirPath() string                { return c.fileDir }
func (c *importCtx) SingleFile() bool               { return c.singleFile }

// performFileImports creates a new [Context] for each file
// and performs imports.
func (c *Checker) performFileImports(files []string, fileContexts map[string]*Context) {
	type imported struct {
		module *Module
		err    error
	}
	importCache := make(map[string]*imported)
	ictx := &importCtx{
		target:     c.module.Target,
		importPath: c.module.ImportPath,
		fileDir:    c.module.Path,
		singleFile: c.module.Flags.Has(SingleFileModule),
	}
	for _, fileName := range files {
		fctx := fileContexts[fileName]
		// Perform imports for the file
		for i, stmt := range c.Programs[fileName].Body {
			imp, ok := stmt.(*ast.ImportStatement)
			if !ok {
				fctx.setAttribute(firstStmtIndex, i)
				break
			}

			impPathStr := imports.ImportPath.String(imp.Module)
			// Try to load from cache, or import it fresh and save it to cache
			res, ok := importCache[impPathStr]
			if !ok {
				res = &imported{}
				res.module, res.err = c.importModule(imp.Module, ictx)
				importCache[impPathStr] = res
			}

			// Apply the import or report the error
			if res.err != nil {
				c.reportImportError(fileName, impPathStr, res.err)
				continue
			}
			c.applyImportedModule(res.module, imp, fctx)
		}
	}
}

func (c *Checker) importModule(p imports.ImportPath, impCtx *importCtx) (*Module, error) {
	if c.Options.Importer == nil {
		// Importer not set up
		return nil, &klarerrs.Error{Code: klarerrs.ErrImporterNotFound}
	}
	return c.Options.Importer.Import(p, impCtx)
}

func (c *Checker) applyImportedModule(mod *Module, stmt *ast.ImportStatement, fctx *Context) {
	if stmt.Alias != nil && stmt.Alias.IsDiscard() {
		return // Don't do anything
	}
	if stmt.Wildcard {
		// We have to import the submodules
	}
	ns := mod.ImportPath.Namespace()
	if stmt.Alias != nil {
		ns = stmt.Alias.Name
	}

	// Declare the namespace
	nsObj := NewObject(ns, fctx.File, stmt.Range, c.module, &Namespace{
		Context: mod.Context,
	})
	if existing := fctx.Declare(nsObj); existing != nil {
		c.fileError(redeclaredError(nsObj, existing, true), fctx.File)
	}

	// Declare unqualified imports
	for _, name := range stmt.UnqualifiedImports {
		target := name.Name.Name
		obj := mod.Context.Lookup(target)
		if obj == nil || !obj.public {
			// Not found in the module or private
			err := klarerrs.NewReferenceError(
				klarerrs.ErrExportUndefined, name.Name.Range(), target,
			).
				SetParam("module", mod.ImportPath.String())
			err.Label = fmt.Sprintf(
				"%s doesn't export %s",
				klarerrs.Quote(mod.ImportPath.String()), klarerrs.Quote(target),
			)
			if obj != nil {
				err.Code = klarerrs.ErrNotExported
				err.AddDetail(
					klarerrs.Quote(target)+" was declared here, and it isn't public",
					obj.FilePath(), obj.Range(),
				)
			}
			c.fileError(err, fctx.File)
			continue
		}
		// Use user-declared unqualified import alias
		if !name.Label.IsZero() {
			obj = new(*obj)
			obj.name = name.Label.Name
		}
		// Declare the unqualified import
		if existing := fctx.Declare(obj); existing != nil {
			c.fileError(redeclaredError(obj, existing, true), fctx.File)
		}
	}
}

func (c *Checker) reportImportError(fileName, importPath string, err error) {
}
