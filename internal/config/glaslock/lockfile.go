package glaslock

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/version"
)

// LockfileVersion is the current version of the lockfile format
const LockfileVersion = 1

var ErrUnsupportedLockfileVersion = fmt.Errorf("unsupported lockfile version")

type PackageSource int

const (
	Git PackageSource = iota
	NPM
	Workspace
	Local
)

type PkgHash uint32

type Lockfile struct {
	Version    int
	Klar       *version.Version
	Packages   []*Package
	PackageMap map[PkgHash]*Package
}

type PackageHeader struct {
	Name      string // Always the name of the package from the manifest
	Version   *version.Version
	From      PackageSource
	GitCommit string // The resolved commit number, if from Git
	Hash      PkgHash
}

type Package struct {
	PackageHeader
	// Workspaces that require this package
	For     []string
	DevOnly bool
	Info    PackageInfo
	Deps    []*PackageHeader
}

type PackageInfo interface {
	packageInfo()
}

type NPMInfo struct {
	Registry  string
	Integrity string
}
type WorkspaceInfo struct {
	Dir string // Relative to project root
}

type GitInfo struct {
	RefType glaspack.GitRefKind
	Ref     string
	URL     string
	// Name of subdirectory in 'pkg' if the package came from a monorepo
	Subpath   string
	Integrity string
}
type LocalInfo struct {
	Path string // Project root or directory in `pkg`
}

func (i *NPMInfo) packageInfo()       {}
func (i *WorkspaceInfo) packageInfo() {}
func (i *GitInfo) packageInfo()       {}
func (i *LocalInfo) packageInfo()     {}

func (p *Package) NPMInfo() *NPMInfo             { return p.Info.(*NPMInfo) }
func (p *Package) WorkspaceInfo() *WorkspaceInfo { return p.Info.(*WorkspaceInfo) }
func (p *Package) GitInfo() *GitInfo             { return p.Info.(*GitInfo) }
func (p *Package) LocalInfo() *LocalInfo         { return p.Info.(*LocalInfo) }
