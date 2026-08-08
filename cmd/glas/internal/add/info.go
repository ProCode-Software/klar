package add

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
)

type pkgInfo struct {
	name       string
	version    string
	desc       string
	deps       []string
	url        string
	etc        map[string]string // Other entries to display
	deprecated *deprecation
}

type deprecation struct {
	msg string
	alt []string
}

func showInfo(pkg Package, info *pkgInfo) {
	var src string
	switch pkg.Source() {
	case WorkspaceSource:
		src = ansi.Bit8(213, "workspace")
	case LocalSource:
		src = ansi.Blue("local")
	case GitSource:
		src = ansi.Bit8(215, "git")
	case NPMSource:
		src = ansi.Red("npm")
	}
	var deprecatedWarning string
	if info.deprecated != nil {
		deprecatedWarning = ansi.ColorSprintf(
			ansi.CodeBoldBrightYellow, " %c Deprecated", icons.Warning,
		)
	}
	title := func(s string) string { return ansi.Bold(s) + ansi.BoldDim(":") }
	ansi.TagPrintfln(
		"%s <d>›</d> <**><c!>%s</c!> <d>%s</>%s",
		src, info.name, info.version, deprecatedWarning,
	)
	if info.deprecated != nil {
		ansi.TagPrintfln("<y!><** u>This package is deprecated:</** u> %s</>", info.deprecated.msg)
		if len(info.deprecated.alt) > 0 {
			ansi.ColorPrintln(
				ansi.CodeYellow, "Alternatives:",
				strings.Join(info.deprecated.alt, ", "),
			)
		}
	}
	if info.desc != "" {
		fmt.Print(info.desc, "\n\n")
	}
	// Repository
	var titleLabel string
	switch pkg.Source() {
	case LocalSource, WorkspaceSource:
		titleLabel = "Path"
	case GitSource, NPMSource:
		titleLabel = "Repository"
	}
	if info.url != "" {
		fmt.Println(title(titleLabel), ansi.Magenta(info.url))
	}
	// Deps
	depString := strings.Join(info.deps, ", ")
	depTitle := "Dependencies"
	if len(info.deps) == 0 {
		depString = ansi.Green("None")
	} else {
		depTitle = fmt.Sprintf("Dependencies (%d)", len(info.deps))
	}
	fmt.Println(title(depTitle), depString)
	// Other fields
	for _, name := range slices.Sorted(maps.Keys(info.etc)) {
		if val := info.etc[name]; val != "" {
			fmt.Println(title(name), val)
		}
	}
}

func formatDependency(name, ver string) string {
	name = ansi.Yellow(name)
	if ver != "" {
		return name + " " + ansi.Dim(ver)
	}
	return name
}

func infoFromManifest(man *glaspack.Manifest) *pkgInfo {
	info := &pkgInfo{
		name:    man.Name,
		desc:    man.Description,
		version: man.Version.String(),
		deps:    make([]string, 0, len(man.Dependencies)),
	}
	if man.Deprecated != nil {
		info.deprecated = &deprecation{
			msg: man.Deprecated.Message,
			alt: man.Deprecated.Alternative,
		}
	}
	for _, dep := range man.Dependencies {
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
			continue // Don't show
		case *glaspack.NPMSpecifier:
			depString = formatDependency(dep.Name, dep.Version.String()+" (npm)")
		}
		info.deps = append(info.deps, depString)
	}
	return info
}
