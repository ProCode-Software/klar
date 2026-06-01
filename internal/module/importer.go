package module

import (
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

type ImportContext = analysis.ImportContext

type CompileFunc func(p imports.ImportPath, dir string, t target.Target) (
	*analysis.Module, error,
)

// BaseImporter is the default importer that resolves imports from the local filesystem.
type BaseImporter struct {
	*PackageInfo
	cache   map[target.Target]map[string]*analysis.Module // string = import path
	Compile CompileFunc
}

func NewBaseImporter(pi *PackageInfo, compile CompileFunc) *BaseImporter {
	return &BaseImporter{
		PackageInfo: pi,
		cache:       make(map[target.Target]map[string]*analysis.Module),
		Compile:     compile,
	}
}

func (i *BaseImporter) Import(p imports.ImportPath, ctx ImportContext) (
	mod *analysis.Module, err error,
) {
	if p.IsStdlib() {
		return i.importStdlib(p, ctx)
	} else if ctx.SingleFile() {
		panic("importing from single-file modules should be blocked by typechecker")
	}
	if cached := i.getCachedModule(ctx.Target(), p.String()); cached != nil {
		return cached, nil
	}

	return
}

func (i *BaseImporter) importStdlib(p imports.ImportPath, ctx ImportContext) (
	mod *analysis.Module, err error,
) {
	// Locate the path of the stdlib on disk
	stdDir, err := KlarStdDir()
	if err != nil {
		return nil, newModuleError(klarerrs.ErrImporterError, p, "", err)
	}
	// Make sure the path isn't private, unless we're bootstrapping
	if p.IsPrivate() && !ctx.Internal() {
		return nil, newModuleError(klarerrs.ErrPrivateImport, p, "", nil)
	}
	// Get the actual module path on disk
	modulePath, err := i.locateImportPath(stdDir, p)
	if err != nil {
		return nil, err
	}
	// Compile!
	mod, err = i.Compile(p, modulePath, ctx.Target())
	if err != nil {
		return nil, newModuleError(klarerrs.ErrModuleCompileError, p, modulePath, err)
	}
	i.cacheModule(ctx.Target(), p.String(), mod)
	return mod, nil
}

func newModuleError(code klarerrs.Code,
	p imports.ImportPath, path string, err error,
) *klarerrs.Error {
	return &klarerrs.Error{
		Code: code,
		Info: klarerrs.ModuleErrorInfo{
			ModulePath: path,
			ImportPath: p.String(),
			Err:        err,
		},
	}
}

func (i *BaseImporter) getCachedModule(t target.Target, path string) *analysis.Module {
	if i.cache[t] == nil {
		return nil
	}
	return i.cache[t][path]
}

func (i *BaseImporter) locateImportPath(pkgDir string, p imports.ImportPath) (string, error) {
	var modulePath string
	if _, ok := KlarProjectDirs[p[0]]; ok {
		// cmd, shared, etc.
		modulePath = pkgDir + sep + filepath.Join(p...)
	} else {
		modulePath = pkgDir + sep + SrcDir + sep + filepath.Join(p...)
	}
	if stat, err := os.Stat(modulePath); err != nil || !stat.IsDir() {
		return "", newModuleError(klarerrs.ErrModuleNotFound, p, modulePath, err)
	}
	return modulePath, nil
}

func (i *BaseImporter) cacheModule(t target.Target, path string, mod *analysis.Module) {
	if i.cache[t] == nil {
		i.cache[t] = make(map[string]*analysis.Module)
	}
	i.cache[t][path] = mod
}
