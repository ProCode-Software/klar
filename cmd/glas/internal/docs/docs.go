package docs

import (
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

func Run(p *argparse.Parser) {
	// Check for incompatible flag combinations
	if p.Flag("web").Bool() {
		for _, flag := range [...]string{"generate", "json", "names"} {
			if p.Flag(flag).Set {
				cli.ColorErrorfln(
					"The <c!>--%s</c!> flag can't be set when displaying the web documentation",
					flag,
				)
				cli.Exit(2)
			}
		}
	}
}

var Flags = argparse.NewParser("[module]").
	BoolFlag("web", "Show documentation in your browser", false, "w").
	BoolFlag("generate", "Generate 'klardoc.json' documentation to the project root", false, "g").
	StringFlag("search", "Search for an export within the module", "query", "", "q").
	BoolFlag("json", "Write 'klardoc.json'-format JSON to standard output", false).
	BoolFlag("names", "Display only the names of exported objects in the module", false).
	BoolFlag("private", "Include documentation for private/unexported objects", false)

var LongDescription = `Shows documentation for a given module. The 'module' argument can be a Klar-style import path, a reference to an object within a module (or the current module), or a filesystem path to a module. The documentation contains all of the exported objects in the module, with descriptions generated from comments in the source code.

By default, the documentation is displayed in the terminal. To display a navigatable version of the documentation in your browser, use the '--web' flag. The served documentation is similar to the ` + ansi.Hyperlink("Klar package documentation website", "https://klarlanguage.github.dev/packages") + `. A local server will be started to serve the documentation.

The '--generate' flag can be used to generate a 'klardoc.json' file in the project root, which can be used by external tools to display documentation, and by the Klar package documentation website for users to view documentation for your packages without installing them. It is recommended to generate this file before publishing a new version of your package. The '--generate' flag generates documentation for all packages and modules in the project, therefore the '[module]' argument must not be provided.

If you want the same documentation format, but don't want to write it to disk (or you want to write it to a different path), use the '--json' flag instead. Unlike '--generate', '--json' can generate documentation for a specific module.`
