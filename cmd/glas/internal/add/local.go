package add

import (
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/util/manifest"
)

type localPackage struct {
	path        string
	manifest    *glaspack.Manifest
	isWorkspace bool
}

func (p *localPackage) Source() PackageSource {
	if p.isWorkspace {
		return WorkspaceSource
	}
	return LocalSource
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

func (p *localPackage) Info(ic *installContext) *pkgInfo {
	if subDir := ic.Flag("subdir").String(); subDir != "" {
		p.path = filepath.Join(p.path, subDir)
	}
	// TODO: Wrap errors given by GetPackageInfo
	projInfo := manifest.GetPackageInfo(p.path)
	p.manifest = projInfo.Manifest
	info := infoFromManifest(p.manifest)
	info.url = p.path
	return info
}

func (p *localPackage) Install(ic *installContext) {
}
