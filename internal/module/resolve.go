package module

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const ManifestName = "glas.pack"

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
// or firstManifest is the project's manifest, firstManifest is returned.
func ResolveProjectManifest(firstManifest string) (string, error) {
	var err error
	if filepath.Base(firstManifest) != ManifestName {
		firstManifest, _, err = ResolveManifest(firstManifest)
		if err != nil {
			return firstManifest, err
		}
	}
	if filepath.
}