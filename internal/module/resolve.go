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

const sep = string(filepath.Separator)

// Per Klar Project Structure Spec
const (
	ManifestName  = "glas.pack"
	PackageFolder = "pkg"
	BuildFileName = "klar.build"
)

type ResolvedPackage struct {
	Dir     string
	Modules []*ResolvedModule
}

type ResolvedModule struct {
	Name     string
	Dir      string
	LastMod  string
	Files    []string
	Internal bool
}

func NormalizeNamespace(ns string) (normalized string, isStd bool) {
	isStd = strings.SplitN(ns, ".", 2)[0] == "klar"
	normalized = strings.ReplaceAll(ns, ".", sep)
	return
}

// ProjectRoot returns the path to the root of a Klar project. If from contains
// a folder part of the standard Klar project folders, ProjectRoot returns the parent
// of that folder. Otherwise, ProjectRoot returns from. There may not be a glas.pack
// file present in path.
func ProjectRoot(from string) (string, error) {
	from, err := filepath.Abs(from)
	if err != nil {
		return from, err
	}
	if info, err := os.Stat(from); err == nil && !info.IsDir() {
		from = filepath.Dir(from)
		if info.Name() == ManifestName {
			return from, nil
		}
	}
	if _, err = os.Stat(from + sep + ManifestName); err == nil {
		return from, nil
	}
	current := from
	// Walk up the directory tree
	for {
		name := filepath.Base(current)
		parent := filepath.Dir(current)
		// Stop if we've reached the root
		if current == parent {
			break
		}
		if _, ok := KlarProjectDirs[name]; ok {
			return parent, nil
		}
		current = parent
	}
	// Not found
	return from, nil
}

// PackageRoot returns the package root and project root for a given path.
// If from is inside a pkg folder, the package root is the folder inside pkg,
// and the project root is the parent of pkg. For other Klar project directories,
// the parent is the project root and also the package root.
func PackageRoot(from string) (pkg, project string, err error) {
	from, err = filepath.Abs(from)
	if err != nil {
		return from, from, err
	}
	// Check if a manifest is located in dir
	if info, err := os.Stat(from); err == nil && !info.IsDir() {
		from = filepath.Dir(from)
		if info.Name() == ManifestName {
			return from, from, nil
		}
	}
	if _, err = os.Stat(from + sep + ManifestName); err == nil {
		return from, from, nil
	}
	// Walk up the directory tree
	curr := from
	var pkgDir string
	for {
		name := filepath.Base(curr)
		parent := filepath.Dir(curr)
		// Stop if we've reached the root
		if curr == parent {
			break
		}
		if _, ok := KlarProjectDirs[name]; ok {
			if name == PackageFolder {
				// For pkg folder: the folder inside pkg is the package root
				// and the parent of pkg is the project root
				if pkgDir != "" {
					return pkgDir, parent, nil
				}
				// Directly in pkg folder
				return "", parent, nil
			} else {
				// For other Klar project dirs: parent is both project and package root
				return parent, parent, nil
			}
		}
		// Track the last directory we saw (potential package inside pkg)
		pkgDir = curr
		curr = parent
	}
	// Not found
	return from, from, nil
}

// IsPackage reports whether path is a path to a package, as defined in the Klar
// Project Structure Spec. IsPackage assumes that path is a directory path.
// IsPackage returns an error if [filepath.Abs] fails.
func IsPackage(path string) (bool, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	current := path
	parent := filepath.Dir(current)
	depth := 0
	// Walk up the directory tree
	for current != parent {
		name := filepath.Base(current)
		if name == PackageFolder && depth == 1 {
			// We're one level inside pkg folder - this is a package
			return true, nil
		} else if _, ok := KlarProjectDirs[name]; ok {
			// Found a Klar project directory - not a package
			return false, nil
		}
		current = parent
		parent = filepath.Dir(current)
		depth++
	}
	return true, nil
}
