package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/module/imports"
)

type PackageInfo struct {
	Manifest *glaspack.Manifest
	// Absolute path to the package directory where glas.pack is
	Dir string
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
func NewPackageInfo(dir string, man *glaspack.Manifest) *PackageInfo {
	if man == nil {
		return nil
	}
	return &PackageInfo{
		Manifest:   man,
		Dir:        dir,
		LocalCache: false,
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
	base = pi.GetModuleAlias(base) // Respect user-defined aliases. TODO
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
	case *glaspack.VersionSpecifier:
		_ = d
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

// GetModuleAlias returns the alias for a given module path, if one is defined
// in the manifest. Otherwise, it returns the original path.
// TODO: The manifest can contain aliases for import paths with
// multiple parts, not just the base.
func (pi *PackageInfo) GetModuleAlias(path string) string {
	if pi.Manifest == nil || pi.Manifest.ModuleAliases == nil ||
		pi.Manifest.ModuleAliases[path] == "" {
		return path
	}
	return pi.Manifest.ModuleAliases[path]
}
