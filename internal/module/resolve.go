package module

import (
	"os"
	"path/filepath"
	"strings"
)

// Directories in the root of a project, as defined by the [Klar Project Structure Spec].
//
// [Klar Project Structure Spec]: https://github.com/ProCode-Software/klar/tree/main/docs/ProjectStructure.md
var KlarProjectDirs = map[string]struct{}{
	".klar": {}, "src": {}, "cmd": {}, "shared": {}, "external": {}, "pkg": {},
	"recipes": {}, "scripts": {}, "generated": {}, "dist": {}, "docs": {},
}

var ProjectOnlyDirs = map[string]struct{}{
	".klar": {}, "pkg": {}, "shared": {},
}

const sep = string(filepath.Separator)

// Per Klar Project Structure Spec
const (
	ManifestFile = "glas.pack"
	BuildFile    = "klar.build"

	PkgDir       = "pkg"
	LocalDataDir = ".klar"
	SrcDir       = "src"
	DistDir      = "dist"
	ExternalDir  = "external"
	SharedDir    = "shared"
	CmdDir       = "cmd"
	TestDir      = "test"
)

func IsProjectDir(name string) bool {
	_, ok := KlarProjectDirs[name]
	return ok
}

// Per Project Structure Spec: No more than 4 parts of a module
const MaxModuleDepth = 4

func splitPath(p string) (string, string) {
	parent, base := filepath.Split(p)
	return strings.TrimSuffix(parent, sep), base
}

// PackageRoot returns the package root and project root
// for a given path, following the Klar Project Structure Spec.
func PackageRoot(p string) (pkg, project string) {
	// Check if a manifest is located in dir
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		p = filepath.Dir(p)
		if info.Name() == ManifestFile {
			proj := p
			if maybePkg := DirFast(p); filepath.Base(maybePkg) == PkgDir {
				proj = DirFast(maybePkg)
			}
			return p, proj
		}
	}
	if _, err := os.Stat(p + sep + ManifestFile); err == nil {
		proj := p
		if maybePkg := filepath.Dir(p); filepath.Base(maybePkg) == PkgDir {
			proj = DirFast(maybePkg)
		}
		return p, proj
	}
	// Walk up the directory tree
	curr, prev := filepath.Clean(p), ""
	for {
		parent, name := splitPath(curr)
		// Stop if we've reached the root
		if curr == parent {
			break
		}
		if _, ok := KlarProjectDirs[name]; ok {
			// Parent of 'pkg' guaranteed to be project root
			if name == PkgDir {
				return prev, parent // x/pkg/y -> (x/pkg/y, x, nil)
			}
			// Found the project root
			if _, ok := ProjectOnlyDirs[name]; ok {
				return parent, parent
			}
			// Check if parent is 'pkg' (e.g: x/pkg/y/src)
			if pkgPar, pkg := splitPath(DirFast(parent)); pkg == PkgDir {
				return parent, pkgPar
			}
			return parent, parent
		}
		// Track the last directory we saw (potential package inside pkg)
		prev = curr // Child
		curr = parent
	}
	// Not found
	return p, p
}

// IsPackage reports whether p is a path to a package, as defined by the Klar
// Project Structure Spec. IsPackage assumes that p is a directory path.
func IsPackage(p string) bool {
	if _, err := os.Stat(filepath.Join(p, ManifestFile)); err == nil {
		return true
	}
	var depth int
	var parent, name string
	for {
		// p is a package if a package directory is found
		parent, name = filepath.Split(p)
		switch {
		case name == PkgDir:
			// We're one level inside pkg folder - this is a package
			return depth == 1
		case IsProjectDir(name):
			// Found a Klar project directory - not a package (parent is)
			return false
		case p == parent:
			return true
		}
		p = parent
		depth++
	}
}

// DirFast is [filepath.Dir] without running [filepath.Clean] on the result.
func DirFast(path string) string {
	vol := filepath.VolumeName(path)
	i := len(path) - 1
	for i >= len(vol) && !os.IsPathSeparator(path[i]) {
		i--
	}
	dir := path[len(vol) : i+1]
	if dir == "." && len(vol) > 2 {
		// must be UNC
		return vol
	}
	return vol + dir
}
