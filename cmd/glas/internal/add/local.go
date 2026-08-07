package add

import (
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util/manifest"
	"github.com/ProCode-Software/klar/internal/version"
)

type localPackage struct {
	path        string
	manifest    *glaspack.Manifest
	isWorkspace bool
}

func (p *localPackage) Source() pkgSource {
	if p.isWorkspace {
		return workspaceSource
	}
	return localSource
}

func (p *localPackage) Name() string {
	if p.manifest != nil {
		return p.manifest.Name
	}
	return p.path
}

func (p *localPackage) ResolvedVersion() string {
	return p.manifest.Version.String()
}

func (p *localPackage) Specifier() (_ version.Specifier)         { panic("invalid usage") }
func (p *localPackage) setNameAndSpec(string, version.Specifier) { panic("invalid usage") }

func (p *localPackage) Info(ic *installContext) *pkgInfo {
	// TODO: Wrap errors given by GetPackageInfo
	projInfo := manifest.GetPackageInfo(p.path)
	man := projInfo.Manifest
	info := &pkgInfo{
		name:    man.Name,
		desc:    man.Description,
		url:     projInfo.Dir,
		version: man.Version.String(),
		deps:    make([]string, len(man.Dependencies)),
	}
	if man.Deprecated != nil {
		info.deprecated = &deprecation{
			msg: man.Deprecated.Message,
			alt: man.Deprecated.Alternative,
		}
	}
	for i, dep := range man.Dependencies {
		var depString string
		switch dep := dep.DependencySpecifier.(type) {
		case *glaspack.GitSpecifier:
			name := dep.URL
			if dep.Subpackage != "" {
				name += " > " + dep.Subpackage
			}
			depString = formatDependency(name, dep.Version.String())
		case *glaspack.LocalSpecifier:
			depString = formatDependency(dep.Path, "")
		case *glaspack.WorkspaceSpecifier:
			depString = formatDependency(filepath.Join(
				projInfo.ProjectDir, module.PkgDir, dep.Subpackage,
			), "")
		case *glaspack.NPMSpecifier:
			depString = formatDependency(dep.Name, dep.Version.String())
		}
		info.deps[i] = depString
	}
	return info
}
func (p *localPackage) Install(ic *installContext) {}
