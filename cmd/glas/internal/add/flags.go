package add

import "github.com/ProCode-Software/klar/pkg/argparse"

var Flags = argparse.NewParser("[packages...]").
	ListFlagOf("for", "The subpackages to add the dependency to", "pkgs", argparse.TypeString, nil, nil, "F").
	BoolFlag("project", "Install for the entire project", false, "P").
	// BoolFlag("npm", "Install packages from NPM", false, "n").
	StringFlag("subdir", "The subdirectory where the dependency is located", "dir", "", "p").
	BoolFlag("dev", "Install as a dev dependency", false, "d").
	BoolFlag("yes", "Install without prompting", false, "y").
	BoolFlag("global", "Install as a global command rather than into the project", false, "g").
	BoolFlag("dry-run", "Don't actually add into the project", false).
	// BoolFlag("force", "?", false, "") // Don't know what this will do yet
	BoolFlag("verbose", "Show verbose output", false, "v").
	// BoolFlag("quiet", "Don't show output while installing (also applies '-y')", false, "q").
	BoolFlag("postinstall", "Run postinstall scripts when installing NPM packages", false).
	BoolFlag("run", "Run the installed command", false, "x").
	BoolFlag("test", "Run the dependencies' tests after installing", false)

var LongDescription = `Installs a package as a dependency. Glas can install packages from Git repositories, NPM, and on the local filesystem. As Glas doesn't have a registry, packages must be installed from a Git repository, such as one hosted on GitHub.

The syntax below can be used to specify where each package should be installed from. To install a package from:

 - A Git repository: '<url>'
 - A package from NPM: 'npm:<package>' (all NPM syntax is supported after 'npm:')
 - A local package: './<dir>'
 - A package in the project's workspace: './pkg/<subpackage>'`
