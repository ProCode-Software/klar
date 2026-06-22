package analysis

import (
	"fmt"
	"slices"
	"strings"

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

// performImports creates a new [Context] for each file and performs imports.
func (c *Checker) performImports(files []string, fileContexts map[string]*Context) {
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
				// Ensure the module has any public exports, otherwise report an error
				if res.err == nil && !res.module.HasExports() {
					res.err = klarerrs.ImportError(klarerrs.ErrNoPublicExports, imp.Module, nil)
				}
				importCache[impPathStr] = res
			}

			// Apply the import or report the error
			if res.err != nil {
				c.reportImportError(impPathStr, res.err, fctx.File, imp)
				c.declareErrorImport(imp, fctx)
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
	c.declare(fctx, nsObj)

	// Declare unqualified imports
	for _, name := range stmt.UnqualifiedImports {
		target := name.Name.Name
		obj := mod.Context.Lookup(target)
		if obj == nil || !obj.public {
			// Not found in the module or private
			err := klarerrs.ReferenceError(
				klarerrs.ErrExportUndefined, name.Name.Range(), target,
			).SetParam("module", mod.ImportPath.String())
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
		// Clone the object so
		// - We can modify its declared range, so when an error is reported for
		// 	its name being redeclared, it shows the range of the import rather
		// 	than the definition outside the current module.
		// - We can use the user-declared unqualified import alias
		obj = new(*obj)
		obj.rang = name.Name.Range()
		obj.file = fctx.File
		obj.module = c.module
		// TODO: Should we change obj's order, and context?
		if !name.Label.IsZero() {
			obj.name = name.Label.Name
		}
		// Declare the unqualified import
		// TODO: Use a custom redeclared error to show "X was already imported"
		c.declare(fctx, obj)
	}
}

func (c *Checker) declareErrorImport(stmt *ast.ImportStatement, fctx *Context) {
	// TODO: declare the namespace with [InvalidType]
}

func (c *Checker) reportImportError(importPath string, err error,
	fid FileID, stmt *ast.ImportStatement,
) {
	kerr, ok := err.(*klarerrs.Error)
	if !ok {
		kerr = &klarerrs.Error{
			Code: klarerrs.ErrImporterError,
			Info: klarerrs.ModuleErrorInfo{ImportPath: importPath, Err: err},
		}
	} else {
		kerr = new(*kerr) // Copy the importer's error so we can add location info
	}
	kerr.Node = stmt
	kerr.Range = stmt.Range

	// Helpful error label
	importPathBase, _, _ := strings.Cut(importPath, ".")
	quotedImportPath := klarerrs.Quote(importPath)
	switch {
	case kerr.Code == klarerrs.ErrNoPublicExports:
		kerr.Label = quotedImportPath + " doesn't provide anything to import"
	case kerr.Code != klarerrs.ErrModuleNotFound:
		// Keep the provided label or use no label
	case slices.Contains(imports.StdlibImports, importPathBase):
		kerr.Label = quotedImportPath + " isn't in the standard library"
	case importPathBase == c.module.ImportPath[0]:
		kerr.Label = quotedImportPath + " isn't in the current package"
	default:
		kerr.Label = "Can't find " + quotedImportPath
	}

	c.fileError(kerr, fid)
}
