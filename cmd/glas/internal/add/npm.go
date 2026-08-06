package add

import (
	"cmp"
	"encoding/json/v2"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/pm/npm"
	"github.com/ProCode-Software/klar/internal/util"
)

var npmRegistry = glaslock.DefaultNPMRegistry

func init() {
	if registry := os.Getenv("NPM_CONFIG_REGISTRY"); registry != "" {
		npmRegistry = registry
	}
}

func splitVersion(name string) (name_, version string, hasVer bool) {
	withoutScopeAt, hasScope := strings.CutPrefix(name, "@")
	name, version, hasVer = strings.Cut(withoutScopeAt, "@")
	if hasScope {
		name = "@" + name
	}
	return name, version, hasVer
}

func (ic *installContext) getNPMInfo(pkg pkg) *pkgInfo {
	// Split the name from the version
	// 	@proicons/react@v4 -> @proicons/react
	//  lodash@v8 -> lodash
	name, version, _ := splitVersion(pkg.name)
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
	if res.StatusCode > 299 {
		var registryHint string
		if npmRegistry != glaslock.DefaultNPMRegistry {
			registryHint = " (" + registryPath + ")"
		}
		if res.StatusCode == 404 {
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
		cli.Failuref(
			"Can't find version %s for NPM package %s",
			klarerrs.Quote(cmp.Or(actualVersion, distTag)),
			klarerrs.Quote(data.Name),
		)
	}

	info := &pkgInfo{
		name:    data.Name,
		version: data.DistTags[distTag],
		desc:    pkgJSON.Description,
		etc: map[string]string{
			"License":   ansi.Green(pkgJSON.License),
			"Published": data.Time[actualVersion].Local().Format(time.DateTime),
			"Keywords": util.JoinColorFunc(
				pkgJSON.Keywords, ansi.CodeMagenta, func(kw string) string {
					// Make the keywords clickable
					return ansi.Hyperlink(kw, "https://www.npmjs.com/search?q=keyword:"+kw)
				}, ", ",
			),
		},
		deps: make([]string, 0, len(pkgJSON.Dependencies)),
	}
	// Dependencies
	for name, ver := range pkgJSON.Dependencies {
		info.deps = append(info.deps, ansi.Yellow(name)+" "+ansi.Dim(ver))
	}

	// Deprecation
	if pkgJSON.Deprecated != "" {
		info.deprecated = &deprecation{msg: pkgJSON.Deprecated}
	}
	return info
}
