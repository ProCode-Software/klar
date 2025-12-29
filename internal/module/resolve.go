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
	GeneratedDir = "generated"
	CmdDir       = "cmd"
)

func IsProjectDir(name string) bool {
	_, ok := KlarProjectDirs[name]
	return ok
}

func splitPath(p string) (string, string) {
	parent, base := filepath.Split(p)
	return strings.TrimSuffix(parent, sep), base
}

// PackageRoot returns the theoretical package root and project root
// for a given path, following the Klar Project Structure Spec.
// PackageRoot returns an error if [filepath.Abs](p) fails.
func PackageRoot(p string) (pkg, project string, err error) {
	// TODO: should we Clean the return paths
	p, err = filepath.Abs(p)
	if err != nil {
		return "", "", err
	}
	// Check if a manifest is located in dir
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		p = filepath.Dir(p)
		if info.Name() == ManifestFile {
			return p, "", nil
		}
	}
	if _, err = os.Stat(p + sep + ManifestFile); err == nil {
		return p, "", nil
	}
	// Walk up the directory tree
	curr, prev := p, ""
	for {
		parent, name := splitPath(curr)
		// Stop if we've reached the root
		if curr == parent {
			break
		}
		if _, ok := KlarProjectDirs[name]; ok {
			// Parent of 'pkg' guaranteed to be project root
			if name == PkgDir {
				return prev, parent, nil // x/pkg/y -> (x/pkg/y, x, nil)
			}
			// Found the project root
			if _, ok := ProjectOnlyDirs[name]; ok {
				return parent, parent, nil
			}
			// Check if parent is 'pkg' (e.g: x/pkg/y/src)
			if pkgPar, pkg := splitPath(filepath.Dir(parent)); pkg == PkgDir {
				return parent, pkgPar, nil
			}
			return parent, parent, nil
		}
		// Track the last directory we saw (potential package inside pkg)
		prev = curr // Child
		curr = parent
	}
	// Not found
	return p, p, nil
}

// IsPackage reports whether p is a path to a package, as defined by the Klar
// Project Structure Spec. IsPackage assumes that p is a directory path.
// IsPackage returns an error if [filepath.Abs] fails.
func IsPackage(p string) (bool, error) {
	p, err := filepath.Abs(p)
	if err != nil {
		return false, err
	}
	var depth int
	var parent, name string
	for {
		// p is a package if a package directory is found
		parent, name = filepath.Split(p)
		switch {
		case name == PkgDir:
			// We're one level inside pkg folder - this is a package
			return depth == 1, nil
		case IsProjectDir(name):
			// Found a Klar project directory - not a package (parent is)
			return false, nil
		case p == parent:
			return true, nil
		}
		p = parent
		depth++
	}
}
