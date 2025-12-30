package build

import (
	"github.com/ProCode-Software/klar/pkg/argparse"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/target"
)

var (
	targetList = map[string]any{
		"js":      target.JavaScript,
		"klarvm":  target.KlarVM,
		"browser": target.Browser,
		"node":    target.Node,
		"deno":    target.Deno,
		"bun":     target.Bun,
	}
	bundleModes = map[string]any{
		"off":    klarbuild.BundleOff,
		"on":     klarbuild.BundleSource,
		"module": klarbuild.BundlePerModule,
		"std":    klarbuild.BundleStd,
	}
	moduleFormats = map[string]any{
		"esm": klarbuild.ModuleESM,
		"umd": klarbuild.ModuleUMD,
	}
)

var Flags = argparse.NewParser("[inputs...]").
	BoolFlag("verbose", "Enable verbose build progress", false, "v").
	BoolFlag("watch", "Rebuild the project when the files are modified", false, "w").
	StringFlag("output", "The directory or file to write output to", "path", "dist", "o").
	EnumFlag("target", "The JavaScript runtime to target", "target", targetList, "", "t").
	StringFlag("banner", "Text to add at the top of each built file", "content", "").
	EnumFlag("bundle", "How to bundle JavaScript output files", "mode", bundleModes, "off").
	BoolFlag("declaration", "Whether TypeScript declaration files should be generated", true).
	StringFlag("declaration-path", "The folder type declarations should be created in", "dir", "").
	BoolFlag("minify", "Whether to minify JavaScript output", false).
	BoolFlag("sourcemap", "Whether to generated sourcemaps for debugging", true).
	BoolFlag("inline-sourcemap", "Whether to generated inline sourcemaps for debugging", false).
	BoolFlag("jsdoc", "Whether to generate JSDoc comments in JavaScript output. Recommended if '--declaration' is disabled", false).
	StringFlag("config", "Path to a klar.build config file", "file", "klar.build", "c").
	BoolFlag("copy-node-modules", "Whether to copy node_modules and package.json to the output directory", true).
	EnumFlag("format", "The JavaScript module format to use", "format", moduleFormats, "esm").
	BoolFlag("json-output", "Show logs and error messages in JSON format", false)
