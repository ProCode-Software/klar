package add

import "github.com/ProCode-Software/klar/pkg/argparse"

var Flags = argparse.NewParser("[packages...]").
	ListFlag("for", "The subpackages to add the dependency to", "pkgs", nil, "F").
	BoolFlag("npm", "Install packages from NPM", false, "n").
	StringFlag("subdir", "The subdirectory where the dependency is located", "dir", "", "p").
	BoolFlag("dev", "Install as a dev dependency", false, "d").
	BoolFlag("yes", "Install without prompting", false, "y").
	BoolFlag("global", "Install as a global command rather than into the project", false, "g").
	BoolFlag("dry-run", "Don't actually add into the project", false, "").
	// BoolFlag("force", "?", false, "") // Don't know what this will do yet
	BoolFlag("verbose", "Show verbose output", false, "v").
	BoolFlag("quiet", "Don't show output while installing (also applies '-y')", false, "q").
	BoolFlag("postinstall", "Run postinstall scripts when installing NPM packages", false).
	BoolFlag("run", "Run the installed command", false, "x").
	BoolFlag("test", "Test the dependencies after installing", false)

var LongDescription = ""
