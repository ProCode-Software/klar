package glaslock

import (
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/version"
)

type PackageSource int

const (
	Git PackageSource = iota
	NPM
	Workspace
	Local
)

type Lockfile struct {
	Version  int
	Klar     *version.Version
	Packages []*Package
}

type PackageHeader struct {
	Path      string
	Version   *version.Version
	From      PackageSource
	GitCommit string // The resolved commit number, if from Git
}

type Package struct {
	PackageHeader
	// Workspaces that require this package
	For       []string
	Deps      []*PackageHeader
	Integrity string // Unless local
	Info      PackageInfo
}

type PackageInfo interface {
	packageInfo()
}

type NPMInfo struct {
	Registry string
}
type WorkspaceInfo struct {
	Dir string // Relative to project root
}

type GitInfo struct {
	RefType glaspack.GitRefKind
	Ref     string
}


func (i NPMInfo) packageInfo() {}
func (i WorkspaceInfo) packageInfo() {}
func (i GitInfo) packageInfo() {}
