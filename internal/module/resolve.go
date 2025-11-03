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
	parts := split(from)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if _, ok := KlarProjectDirs[part]; ok {
			// TODO: use filepath.Dir()
			return filepath.Join(parts[:i]...), nil
		}
	}
	// Return the current directory if a project wasn't found
	return from, nil
}

// PackageRoot returns the folder where
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
	pkg, project = projectRootFast(from)
	return
}

// [PackageRoot] without calling [os.Stat]
func projectRootFast(from string) (pkg, project string) {
	parts := split(from)
	// Loop backwards for closest package
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part == "pkg" {
			project = filepath.Join(parts[:i]...)
			subPackage := strings.Join([]string{project, part, parts[i+1]}, sep)
			return subPackage, project
		} else if _, ok := KlarProjectDirs[part]; ok {
			project = filepath.Join(parts[:i]...)
			return project, project
		}
	}
	// Return the current directory if a project wasn't found
	return from, from
}

// IsPackage reports whether path is a path to a package, as defined in the Klar
// Project Structure Spec. IsPackage assumes that path is a directory path.
// IsPackage returns an error if [filepath.Abs] fails.
func IsPackage(path string) (bool, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	parts := split(path)
	for i := len(parts) - 1; i >= 0; i-- {
		dir := parts[i]
		if dir == PackageFolder && i == len(parts)-2 {
			return true, nil
		} else if _, ok := KlarProjectDirs[dir]; ok {
			return false, nil
		}
	}
	return true, nil
}

// Implement a cache system to avoid resplitting paths.
var segmentCache = map[string][]string{}

func split(path string) []string {
	if cached, ok := segmentCache[path]; ok {
		return cached
	}
	segments := strings.Split(path, sep)
	segmentCache[path] = segments
	return segments
}

func SplitSegments(path string) []string {
	return split(path)
}
