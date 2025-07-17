package build

import (
	"fmt"
	"path/filepath"

	"github.com/ProCode-Software/klar/cmd/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/target"
)

func Build(r *command.Runner) {
	projDir := r.Arg(1)
	b := &build.Compiler{
		Mode: build.ModeBuild,
		Options: &build.Options{
			JS:      &build.JSOptions{},
			Verbose: true,
		},
	}
	b.Log("Starting build...")
	manifest := cli.ResolveManifest(projDir)
	b.LogDone("Manifest found at " + manifest)
	projDir = filepath.Dir(manifest)
	b.Options.ProjectDir = projDir

	ParseFlags(r, b.Options)
	b.LogDone("Done parsing command line flags")
	fmt.Println(manifest)
}

var jsBoolFlags = map[string]build.Flags{
	"declaration":       build.CreateDeclaration,
	"minify":            build.Minify,
	"sourcemap":         build.CreateSourceMap,
	"jsdoc":             build.CreateJSDoc,
	"copy-node-modules": build.CopyNodeModules,
}

func ParseFlags(r *command.Runner, o *build.Options) {
	var firstJSFlag string
	for flag, v := range r.AllFlags() {
		if v == nil {
			continue
		}
		if flag, ok := jsBoolFlags[flag]; ok && v == true {
			o.JS.Flags |= flag
		}
		switch flag {
		case "config":
			continue
		case "verbose":
			o.Verbose = v.(bool)
			continue
		case "watch":
			o.Watch = v.(bool)
			continue
		case "output":
			o.OutputDir = v.(string)
			continue
		case "target":
			o.Target = v.(target.Double)
			continue

		// JavaScript options
		case "banner":
			o.JS.Banner = v.(string)
		case "bundle":
			o.JS.Bundle = v.(js.BundleMode)
		case "declaration-dir":
			o.JS.DeclarationDir = v.(string)
		case "format":
			o.JS.Format = v.(js.ModuleFormat)
		default:
			panic("unhandled flag: " + flag)
		}
		if firstJSFlag == "" && !r.IsDefault(flag) {
			firstJSFlag = flag
		}
	}
	// Check if a JavaScript flag was used when not targeting JavaScript
	if t := o.Target.Target; t != target.JavaScript && firstJSFlag != "" {
		cli.Failure(fmt.Sprintf(
			"Can't use JavaScript flag '--%s' with target '%s'", firstJSFlag, t,
		))
	}
}
