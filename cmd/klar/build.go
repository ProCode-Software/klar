package main

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/target"
)

var (
	targetList = map[string]any{
		"node":    target.Double{target.JavaScript, target.JSNode},
		"deno":    target.Double{target.JavaScript, target.JSDeno},
		"bun":     target.Double{target.JavaScript, target.JSBun},
		"browser": target.Double{target.JavaScript, target.JSBrowser},
	}
	bundleModes = map[string]any{
		"off":    js.BundleOff,
		"on":     js.BundleSource,
		"module": js.BundlePerModule,
		"std":    js.BundleStd,
	}
	moduleFormats = map[string]any{
		"esm": js.ModuleESM,
		"umd": js.ModuleUMD,
	}
)

func RunBuild() {
	cmd := cli.NewArgParser(Commands["build"], 1).
		BoolFlag("verbose", "Enable verbose build progress", false, "v").
		BoolFlag("watch", "Rebuild the project when the files are modified", false, "w").
		StringFlag("output", "The directory or file to write output to", "dist", "o").
		OptionFlag("target", "The JavaScript runtime to target", targetList, "browser", "t").
		StringFlag("banner", "Text to add at the top of each built file", "").
		OptionFlag("bundle", "How to bundle JavaScript output files", bundleModes, "off").
		BoolFlag("declaration", "Whether TypeScript declaration files should be generated", true).
		StringFlag("declaration-dir", "The folder type declarations should be created in", "").
		BoolFlag("minify", "Whether to minify JavaScript output", false).
		BoolFlag("sourcemap", "Whether to generated sourcemaps for debugging", true).
		BoolFlag("jsdoc", "Whether to generate JSDoc comments in JavaScript output. Recommended if '--declaration' is disabled", false).
		StringFlag("config", "Path to a klar.build config file", "klar.build", "c").
		BoolFlag("copy-node-modules", "Whether to copy node_modules and package.json to the output directory", true).
		OptionFlag("format", "The JavaScript module format to use", moduleFormats, "esm")

	cmd.Parse()
	projDir := cmd.ArgAt(1)
	if projDir == "" {
		cwd, err := os.Getwd()
		handleErr(err)
		projDir = cwd
	}
	manifest := ResolveManifest(projDir)
	fmt.Println(manifest)
}
