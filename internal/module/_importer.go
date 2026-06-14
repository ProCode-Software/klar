package module

import (
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

type (
	ImportContext = analysis.ImportContext

	// CompileFunc is the function signature for compiling a module.
	// Implemented by [build.Compiler].
	CompileFunc func(ctx ImportContext, dir string) (*analysis.Module, error)

	// Target-specific cached modules. The [string] key = filesystem path
	importCache map[target.Target]map[string]*analysis.Module
)

// BaseImporter is the default importer that resolves imports from the
// local filesystem.
type BaseImporter struct {
	*PackageInfo
	cache   importCache
	Compile CompileFunc

	// TODO: Honor the manifest's import aliases, which may also include
	// more than just the base of the import path
}

func NewBaseImporter(pi *PackageInfo, compile CompileFunc) *BaseImporter {
	return &BaseImporter{
		PackageInfo: pi,
		cache:       make(importCache),
		Compile:     compile,
	}
}

func (i *BaseImporter) Import(p imports.ImportPath, ctx ImportContext) (
	mod *analysis.Module, err error,
) {
	var modulePath string
	switch {
	case p.IsStdlib(): // TODO: there may be an alias to the std
		// Single-file modules can import the standard library (only)
		if modulePath, err = i.getStdDir(p, ctx); err != nil {
			return nil, err
		}
	case ctx.SingleFile():
		// The type checker should already be raising an error before calling Import
		panic("importing from single-file modules should be blocked by typechecker")
	default:
		// Compile a project module or dependency
		//
		// Make the map of base import paths
		if err := i.MakeModuleMap(); err != nil {
			return nil, err // TODO: return a different error type?
		}
		// Find the filesystem path based on the import path base
		dir, found, conflict := i.getDirFromBase(p[0])
		switch {
		case conflict:
			return nil, klarerrs.ImportError(klarerrs.ErrImportPathConflict, p, dir, nil)
		case !found:
			return nil, klarerrs.ImportError(klarerrs.ErrModuleNotFound, p, "", nil)
		default:
			modulePath = dir
		}
		// TODO: check privacy after resolving aliases
	}
	// Check if the module is already cached
	if cached := i.getCachedModule(ctx.Target(), modulePath); cached != nil {
		return cached, nil
	}
	// Compile if it isn't
	mod, err = i.Compile(ctx, modulePath)
	if err != nil {
		return nil, klarerrs.ImportError(klarerrs.ErrModuleCompileError, p, modulePath, err)
	}
	i.cacheModule(ctx.Target(), modulePath, mod) // Save so we don't do this again
	return mod, nil
}

func (i *BaseImporter) getStdDir(p imports.ImportPath, ctx ImportContext) (string, error) {
	// Locate the path of the stdlib on disk
	stdDir, err := KlarStdDir()
	if err != nil {
		return "", klarerrs.ImportError(klarerrs.ErrImporterError, p, "", err)
	}
	// Make sure the path isn't private, unless we're bootstrapping
	if p.IsPrivate() && !ctx.Internal() {
		return "", klarerrs.ImportError(klarerrs.ErrPrivateImport, p, "", nil)
	}
	// Get the actual module path on disk
	return i.locateImportPath(stdDir, p)
}

// locateImportPath resolves the import path relative to a given package
// directory. It returns the filesystem path of the module, or an error if
// the path doesn't exist or is not a directory (single-file scripts can't be imported).
func (i *BaseImporter) locateImportPath(pkgDir string, p imports.ImportPath) (string, error) {
	var modulePath string
	if _, ok := KlarProjectDirs[p[0]]; ok {
		// cmd, shared, etc.
		modulePath = pkgDir + sep + filepath.Join(p...)
	} else {
		modulePath = pkgDir + sep + SrcDir + sep + filepath.Join(p...)
	}
	if stat, err := os.Stat(modulePath); err != nil || !stat.IsDir() {
		return "", klarerrs.ImportError(klarerrs.ErrModuleNotFound, p, modulePath, err)
	}
	return modulePath, nil
}

func (i *BaseImporter) getCachedModule(t target.Target, path string) *analysis.Module {
	if i.cache[t] == nil {
		return nil
	}
	return i.cache[t][path]
}

func (i *BaseImporter) cacheModule(t target.Target, path string, mod *analysis.Module) {
	if i.cache[t] == nil {
		i.cache[t] = make(map[string]*analysis.Module)
	}
	i.cache[t][path] = mod
}
