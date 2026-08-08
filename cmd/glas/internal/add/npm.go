package add

import (
	"cmp"
	"encoding/json/v2"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/pm/npm"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/internal/version"
)

var npmRegistry = glaslock.DefaultNPMRegistry

func init() {
	if registry := os.Getenv("NPM_CONFIG_REGISTRY"); registry != "" {
		npmRegistry = registry
	}
}

var _ Package = &npmPackage{}

type npmPackage struct {
	nameAndVersion string
	manifest       *npm.RegistryVersion
}

func (p *npmPackage) Source() glaslock.PackageSource { return NPMSource }
func (p *npmPackage) Name() string {
	if p.manifest == nil {
		name, _, _ := splitVersion(p.nameAndVersion)
		return name
	}
	return p.manifest.Name
}
func (p *npmPackage) ResolvedVersion() string { return p.manifest.Version }
func (p *npmPackage) Info(ic *installContext) *pkgInfo {
	// Split the name from the version
	// 	@proicons/react@v4 -> @proicons/react
	//  lodash@v8 -> lodash
	//
	// TODO: NPM allows local paths and protocols. Don't fail if not a valid URL
	name, version, _ := splitVersion(p.nameAndVersion)
	registryPath, err := url.JoinPath(npmRegistry, name)
	if err != nil {
		cli.Failure("Invalid URL:", err)
	}
	res, err := http.Get(registryPath)
	if err != nil {
		cli.FailureDetailf(
			"Failed to fetch %q from NPM registry: ", "%v", name, err.Error(),
		)
	}
	defer res.Body.Close()
	var registryHint string
	if npmRegistry != glaslock.DefaultNPMRegistry {
		registryHint = " (" + registryPath + ")"
	}
	if res.StatusCode > 299 {
		if res.StatusCode == http.StatusNotFound {
			cli.Failuref(
				"Package %s not found in NPM registry%s",
				klarerrs.Quote(name), registryHint,
			)
		}
		cli.FailureDetailf(
			"Error while getting package %s from NPM registry%s: ", res.Status,
			klarerrs.Quote(name), registryHint,
		)
	}

	var data npm.RegistryData
	if err := json.UnmarshalRead(res.Body, &data); err != nil {
		cli.FailureDetailf(
			"Failed to parse data from NPM registry for package %q: ", "%v",
			name, err.Error(),
		)
	}
	// If the package was unpublished, there will be an "unpublished" field
	// in .times (that is an object)
	// TODO: Will this check fail if any, but not all, versions were unpublished?
	if !data.Time["unpublished"].IsZero() {
		cli.Failuref(
			"Package %s was unpublished from NPM %s",
			klarerrs.Quote(name), registryHint,
		)
	}
	// On the CLI, the version can be exact, or a tag like 'beta'
	// TODO: Either Klar or NPM specifiers may be allowed
	distTag := "latest"
	if _, ok := data.DistTags[version]; ok {
		distTag = version
	}
	actualVersion := data.DistTags[distTag]
	if _, ok := data.Versions[version]; ok {
		actualVersion = version
	}
	pkgJSON := data.Versions[actualVersion]
	if pkgJSON == nil {
		if len(data.Versions) == 0 {
			cli.Failuref(
				"NPM package %s has no versions available", klarerrs.Quote(data.Name),
			)
		}
		cli.Failuref(
			"Can't find version %s for NPM package %s",
			klarerrs.Quote(cmp.Or(actualVersion, distTag)),
			klarerrs.Quote(data.Name),
		)
	}
	return p.infoFromRegistry(&data, distTag, actualVersion)
}

func (p *npmPackage) infoFromRegistry(data *npm.RegistryData, distTag, actualVersion string) *pkgInfo {
	pkgJSON := data.Versions[actualVersion]
	info := &pkgInfo{
		name:    data.Name,
		version: "v" + data.DistTags[distTag],
		desc:    pkgJSON.Description,
		etc: map[string]string{
			"License":   ansi.Green(pkgJSON.License),
			"Published": data.Time[actualVersion].A().Local().Format(time.DateTime),
			"Keywords": util.JoinColorFunc(
				pkgJSON.Keywords, ansi.CodeMagenta, func(kw string) string {
					// Make the keywords clickable
					return ansi.Hyperlink(kw, "https://www.npmjs.com/search?q=keyword:"+kw)
				}, ", ",
			),
		},
		deps: make([]string, 0, len(pkgJSON.Dependencies)),
	}
	if pkgJSON.License == "" {
		info.etc["License"] = ansi.Gray("Unknown")
	}
	if pkgJSON.Dist.UnpackedSize > 0 {
		info.etc["Size (without deps)"] = util.FormatSize(pkgJSON.Dist.UnpackedSize)
	}

	// Dependencies
	depString := func(name, ver string) string {
		name = ansi.Hyperlink(ansi.Yellow(name), "https://npmjs.com/package/"+name)
		return formatDependency(name, ver)
	}
	for name, ver := range pkgJSON.Dependencies {
		info.deps = append(info.deps, depString(name, ver))
	}
	for name, ver := range pkgJSON.PeerDependencies {
		info.deps = append(info.deps, depString(name, ver))
	}

	// Deprecation
	if pkgJSON.Deprecated != "" {
		info.deprecated = &deprecation{msg: pkgJSON.Deprecated}
	}

	// Monthly downloads
	if downloads, err := p.getMonthlyDownloads(data.Name); err == nil {
		info.etc["Monthly downloads"] = util.FormatNumber(downloads)
	}
	return info
}

func (p *npmPackage) getMonthlyDownloads(name string) (int, error) {
	res, err := http.Get("https://api.npmjs.org/downloads/point/last-month/" + name)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	var data struct {
		Downloads int `json:"downloads"`
	}
	if err := json.UnmarshalRead(res.Body, &data); err != nil {
		return 0, err
	}
	return data.Downloads, nil
}

func (p *npmPackage) Install(ic *installContext) {
}

func (p *npmPackage) KlarVersion() version.Version {
	if ver, err := version.Parse(p.ResolvedVersion()); err == nil {
		return ver
	}
	// TODO: Convert SemVer to Klar version
	return version.Version{}
}
