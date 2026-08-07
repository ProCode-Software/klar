package add

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/icons"
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
	case workspaceSource:
		src = ansi.Bit8(213, "workspace")
	case localSource:
		src = ansi.Blue("local")
	case gitSource:
		src = ansi.Bit8(216, "git")
	case npmSource:
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
	// Deps
	depString := strings.Join(info.deps, ", ")
	depTitle := "Dependencies"
	if len(info.deps) == 0 {
		depString = ansi.Green("None")
	} else {
		depTitle = fmt.Sprintf("Dependencies (%d)", len(info.deps))
	}
	fmt.Println(title(depTitle), depString)
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
