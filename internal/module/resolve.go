package module

import (
	"errors"
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

// ResolveManifest returns the nearest glas.pack file from from.
// If a glas.pack file was not found, found == false and err == nil.
// If another error occured while looking for a manifest, found == false
// and err == the error that occured.
func ResolveManifest(from string) (path string, found bool, err error) {
	from, err = filepath.Abs(from)
	if err != nil {
		return "", false, err
	}
	if info, err := os.Stat(from); err != nil {
		return "", false, err
	} else if !info.IsDir() {
		from = filepath.Dir(from)
	}
	var last string
	for {
		manifestPath := filepath.Join(from, ManifestName)
		if _, err := os.Stat(manifestPath); err == nil {
			// Found
			return manifestPath, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			// Other error
			return "", false, err
		}
		last = from
		from = filepath.Dir(from)
		if from == last {
			// Reached root
			return "", false, nil
		}
	}
}

// ResolveProjectManifest resolves the glas.pack file for a project.
// Unlike ResolveManifest, which returns the closest manifest to a given
// path, which could be in a sub-package, ResolveProjectManifest finds
// the full project's manifest, outside the pkg folder. If one is not found,
// or firstPath is the project's manifest, firstPath is returned.
// If firstPath
func ResolveProjectManifest(firstPath string) (string, error) {
	if filepath.Base(firstPath) != ManifestName {
		var err error
		firstPath, _, err = ResolveManifest(firstPath)
		if err != nil {
			return firstPath, err
		}
	}
	firstPath = filepath.Clean(firstPath)
	parts := strings.Split(firstPath, string(filepath.Separator))
	for i, part := range parts {
		if part == PackageFolder {
			return strings.Join(parts[:i], ""), nil
		}
		if i == len(parts) {
			return firstPath, nil
		}
	}
	return firstPath, nil
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
	parts := strings.Split(from, sep)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if _, ok := KlarProjectDirs[part]; ok {
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
	parts := strings.Split(from, sep)
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
