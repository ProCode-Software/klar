package add

import (
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/version"
)

type gitPackage struct {
	url      string
	manifest *glaspack.Manifest
	spec     version.Specifier
}

func (p *gitPackage) Source() pkgSource                { return gitSource }
func (p *gitPackage) ResolvedVersion() string          { return p.manifest.Version.String() }
func (p *gitPackage) Specifier() (_ version.Specifier) { return p.spec }
func (p *gitPackage) Name() string {
	if p.manifest != nil {
		return p.manifest.Name
	}
	return p.url
}

func (p *gitPackage) setNameAndSpec(name string, spec version.Specifier) {
	p.url, p.spec = name, spec
}

func (p *gitPackage) Info(ic *installContext) *pkgInfo { return nil }
func (p *gitPackage) Install(ic *installContext)       {}
