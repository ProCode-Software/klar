package add

import (
	"fmt"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/config/glaslock"
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

func (p *localPackage) Source() glaslock.PackageSource {
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

func (p *localPackage) KlarVersion() version.Version { return p.manifest.Version }
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

func (p *localPackage) workspacePkgName() string {
	if !p.isWorkspace {
		panic("workspacePkgName called on non-workspace package")
	}
	pkgDir, pkgName := filepath.Split(p.path)
	if filepath.Base(pkgDir) != module.PkgDir {
		panic(fmt.Sprintf("expected %s dir, but got %s", module.PkgDir, pkgDir))
	}
	return pkgName
}
