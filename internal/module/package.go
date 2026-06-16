package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/module/imports"
)

type PackageInfo struct {
	Manifest *glaspack.Manifest
	// Absolute path to the package directory where glas.pack is
	Dir string
	// Only different from Dir in workspaces
	ProjectDir string
	// Whether cache is stored in the `.klar` folder in Dir
	LocalCache bool
	// First part of import path : local path
	// `cmd` and `shared` are local to the current package.
	// Example:
	// 	map[string]string{"klar", ".../src/klar", "cmd": ".../cmd"}
	moduleMap map[string]string
	// Conflicting import paths without manifest-defined aliases.
	// First part of import path : package names
	importConflicts map[string][]string
}

// NewPackageInfo creates a new PackageInfo instance. If the manifest is nil,
// NewPackageInfo returns nil.
func NewPackageInfo(pkgDir, projDir string, man *glaspack.Manifest) *PackageInfo {
	if man == nil {
		return nil
	}
	if projDir == "" {
		projDir = pkgDir
	}
	var localCache bool
	if _, err := os.Stat(filepath.Join(projDir, LocalDataDir)); err == nil {
		localCache = true
	}
	return &PackageInfo{
		Manifest:   man,
		Dir:        pkgDir,
		ProjectDir: pkgDir,
		LocalCache: localCache,
		moduleMap:  nil,
	}
}

func (pi *PackageInfo) ImportPathOf(p string) imports.ImportPath {
	var err error
	if p, err = filepath.Rel(pi.Dir, p); err != nil {
		panic(fmt.Sprintf(
			"pi.ImportPathOf(%q): argument is not located in pi.Dir [%s]",
			p, pi.Dir,
		))
	}
	parts := strings.Split(p, string(filepath.Separator))
	switch base := parts[0]; base {
	case CmdDir, SharedDir, TestDir:
		return imports.ImportPath(parts)
	case SrcDir:
		return imports.ImportPath(parts[1:])
	default:
		// TODO: If this could happen, should this function return an error instead?
		panic(fmt.Sprintf("pi.ImportPathOf(%q): invalid base directory %q", p, base))
	}
}

func (pi *PackageInfo) MakeModuleMap() error {
	if pi.moduleMap != nil {
		return nil
	}
	pi.moduleMap = make(map[string]string)
	for _, dep := range pi.Manifest.Dependencies {
		dir, base, err := pi.ResolveDependency(dep.DependencySpecifier)
		if err != nil {
			return err
		}
		pi.addBaseToModuleMap(base, dir)
	}
	// Special directories: Part of Klar base folder structure,
	// and local to the current package.
	// Base import path = directory name
	for _, name := range [...]string{CmdDir, SharedDir, TestDir} {
		dir := filepath.Join(pi.Dir, name)
		if _, err := os.Stat(dir); err == nil {
			pi.addBaseToModuleMap(name, dir)
		}
	}
	return nil
}

func (pi *PackageInfo) addBaseToModuleMap(base, dir string) {
	if firstDir, ok := pi.moduleMap[base]; ok {
		// Import conflict: add existing and new dir to conflict list
		pi.addImportConflict(base, firstDir)
		pi.addImportConflict(base, dir)
		delete(pi.moduleMap, base) // Can't import with base if there's a conflict
	} else if pi.getImportConflict(base) != nil {
		// Existing conflict
		pi.addImportConflict(base, dir)
	}
	pi.moduleMap[base] = dir // No conflict (yet)
}

func (pi *PackageInfo) getDirFromBase(base string) (dir string, found, conflict bool) {
	dir, found = pi.moduleMap[base]
	if found {
		return dir, true, false
	}
	conflict = pi.getImportConflict(base) != nil
	return dir, !conflict, conflict
}

func (pi *PackageInfo) ResolveDependency(d glaspack.DependencySpecifier) (
	dir, importBase string, err error,
) {
	switch d := d.(type) {
	case *glaspack.NPMSpecifier:
	case *glaspack.WorkspaceSpecifier:
	case *glaspack.LocalSpecifier:
	case *glaspack.GitSpecifier:
	default:
		panic(fmt.Sprintf("unhandled specifier: %#v", d))
	}
	return
}

func (pi *PackageInfo) addImportConflict(base, pkg string) {
	if pi.importConflicts == nil {
		pi.importConflicts = make(map[string][]string)
	}
	pi.importConflicts[base] = append(pi.importConflicts[base], pkg)
}

func (pi *PackageInfo) getImportConflict(base string) []string {
	if pi.importConflicts == nil {
		return nil
	}
	return pi.importConflicts[base]
}

func (pi *PackageInfo) CacheDir() string {
	if pi.LocalCache {
		return filepath.Join(pi.ProjectDir, LocalDataDir, "cache")
	}
	return SystemDirs.Cache
}

func (pi *PackageInfo) DataDir() string {
	if pi.LocalCache {
		return filepath.Join(pi.ProjectDir, LocalDataDir)
	}
	return SystemDirs.Data
}

func (pi *PackageInfo) PackageDirOf(p *glaslock.Package) string {
	switch p.From {
	case glaslock.NPM:
		panic("npm packages not supported with PackageDirOf")
	case glaslock.Local:
		path := p.LocalInfo().Path
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(pi.Dir, path)
	case glaslock.Workspace:
		dir := p.WorkspaceInfo().Dir
		return filepath.Join(pi.ProjectDir, PkgDir, dir)
	case glaslock.Git:
		data := p.GitInfo()
		return filepath.Join(
			pi.DataDir(), "packages",
			strings.ReplaceAll(strings.TrimPrefix(data.URL, "https://"), "/", "+"),
			data.Integrity, data.Subpath,
		)
	default:
		panic(fmt.Sprintf("unhandled package source: %v", p.From))
	}
}

// ModuleDirOf returns the directory path of the module with the
// given import path. ModuleDirOf expects p to be part of the package.
// pi.ModuleDirOf is equivalent to [ModuleDirOf](p, pi.Dir, pi.ProjectDir).
func (pi *PackageInfo) ModuleDirOf(p imports.ImportPath) string {
	return ModuleDirOf(p, pi.Dir, pi.ProjectDir)
}

// ModuleDirOf returns the directory path of the module within the package
// with the given import path.
func ModuleDirOf(p imports.ImportPath, packageDir, projectDir string) string {
	switch p[0] {
	case TestDir, CmdDir:
		// Located in the package, but not 'src'
		return packageDir + sep + filepath.Join(p...)
	case SharedDir:
		// 'shared' directory can only be in the PROJECT root
		return projectDir + sep + filepath.Join(p...)
	default:
		// Module located in the src folder of the package
		return packageDir + sep + SrcDir + sep + filepath.Join(p...)
	}
}
