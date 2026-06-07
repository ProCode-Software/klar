package build

import (
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

var Flags = argparse.NewParser("[inputs...]").
	BoolFlag("verbose", "Enable verbose build progress", false, "v").
	BoolFlag("watch", "Rebuild the project when the files are modified", false, "w").
	StringFlag("output", "The directory or file to write output to", "path", "", "o").
	EnumFlag("target", "The JavaScript runtime to target", "target", target.Names, "", "t").
	StringFlag("banner", "Text to add at the top of each built file", "content", "").
	EnumFlag("bundle", "How to bundle JavaScript output files", "mode", klarbuild.BundleModes, "off").
	BoolFlag("declaration", "Whether TypeScript declaration files should be generated", true).
	StringFlag("declaration-path", "The folder type declarations should be created in", "dir", "").
	BoolFlag("minify", "Whether to minify JavaScript output", false).
	BoolFlag("sourcemap", "Whether to generated sourcemaps for debugging", true).
	BoolFlag("inline-sourcemap", "Whether to generated inline sourcemaps for debugging", false).
	BoolFlag("jsdoc", "Whether to generate JSDoc comments in JavaScript output. Recommended if '--declaration' is disabled", false).
	StringFlag("config", "Path to a klar.build config file. By default, it is searched for in the current and parent directories", "file", "", "c").
	BoolFlag("copy-node-modules", "Whether to copy node_modules and package.json to the output directory", true).
	BoolFlag("json-output", "Show logs and error messages in JSON format", false).
	BoolFlag("sound-on-error", "Play a sound when there are errors", false)

var (
	klarBuildFlags = []string{"watch", "output", "target"}
	jsFlags        = []string{
		"declaration", "minify", "inline-sourcemap", "sourcemap", "jsdoc",
		"copy-node-modules", "banner", "bundle", "declaration-path",
	}
)
