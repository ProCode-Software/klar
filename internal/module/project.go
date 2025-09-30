package module

import (
	"os"
	"path/filepath"
)

type ProjectInfo struct {
	ProjectRoot string
	PackageRoot string
	PackageName string
}

func GetProjectInfo(fromPath string) (*ProjectInfo, error) {
	pkg, proj, err := PackageRoot(fromPath)
	if err != nil {
		return nil, err
	}
	return &ProjectInfo{
		PackageRoot: pkg,
		ProjectRoot: proj,
		PackageName: filepath.Base(pkg),
	}, nil
}

// Returns an empty string if no manifest was found.
func (i *ProjectInfo) Manifest() string {
	man := i.ProjectRoot + sep + ManifestName
	if _, err := os.Stat(man); err == nil {
		return man
	}
	man = i.PackageRoot + sep + ManifestName
	if _, err := os.Stat(man); err == nil {
		return man
	}
	return ""
}
