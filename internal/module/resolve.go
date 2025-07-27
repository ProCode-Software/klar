package module

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

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
	normalized = strings.ReplaceAll(ns, ".", string(filepath.Separator))
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
