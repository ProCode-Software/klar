package module

import (
	"cmp"
	"os"
	"path/filepath"
	"strings"
)

// Per Project Structure Spec: No more than 4 parts of a module
const MaxModuleDepth = 4

func splitPath(p string) (string, string) {
	parent, base := filepath.Split(p)
	return strings.TrimSuffix(parent, sep), base
}

// PackageRoot returns the package root and project root
// for a given path, following the Klar Project Structure Spec. For accurate
// results, p should be an absolute path.
func PackageRoot(p string) (pkg, project string) {
	// Walk up the directory tree
	curr, prev := filepath.Clean(p), ""
	var knownPkg string
	for {
		parent, name := splitPath(curr)
		// Stop if we've reached the root
		if curr == parent {
			break
		}
		if _, ok := KlarPackageDirs[name]; ok {
			// Parent of 'pkg' guaranteed to be project root
			if name == PkgDir {
				return prev, parent // x/pkg/y -> (x/pkg/y, x)
			}
			// Found the project root
			if _, ok := ProjectOnlyDirs[name]; ok {
				return cmp.Or(knownPkg, parent), parent
			}
			// Package directory, but may not be the root
			knownPkg = parent
		}
		// Track the last directory we saw (potential package inside pkg)
		prev, curr = curr, parent
	}
	// No Klar-specific directories found. The provided path is the package/project
	pkg = cmp.Or(knownPkg, p)
	return pkg, pkg
}

// IsPackage reports whether p is a path to a package, as defined by the Klar
// Project Structure Spec. IsPackage assumes that p is a directory path.
func IsPackage(p string) bool {
	if _, err := os.Stat(filepath.Join(p, ManifestFile)); err == nil {
		return true
	}
	p = filepath.Clean(p)
	var depth int
	var parent, name string
	for {
		// p is a package if a package directory is found
		parent, name = filepath.Split(p)
		switch {
		case name == PkgDir:
			// We're one level inside pkg folder - this is a package
			return depth == 1
		case IsPackageDir(name):
			// Found a Klar project directory - not a package (parent is)
			return false
		case p == parent:
			return true // No special directories found
		}
		p = strings.TrimSuffix(parent, sep)
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
