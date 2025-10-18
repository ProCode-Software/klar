package build

import (
	"fmt"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/argparse"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/target"
)

func Build(r *command.Runner) {
	projDir := r.Arg(1)
	b := &build.Compiler{
		Mode: build.ModeBuild,
		Options: []*build.Options{
			{JS: &build.JSOptions{}},
		},
		Verbose: r.Flag("verbose") == true,
	}
	if hasLogFile := b.InitLogger(); hasLogFile {
		defer build.KLAR_LOG_FILE.Close()
	}
	b.Println("Starting build...")
	manifest := cli.ResolveManifest(projDir)
	b.Printf("Manifest found at %s\n", manifest)
	projDir = filepath.Dir(manifest)

	ParseFlags(r, b.Options[0])
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
	for flag, v := range r.AllFlags() {
		if v == nil {
			continue
		}
		if flag, ok := jsBoolFlags[flag]; ok && v.Value() == true {
			o.JS.Flags |= flag
			continue
		}
		switch flag {
		case "config":
			continue // TODO
		case "watch":
			o.Watch = v.Value().(bool)
		case "output":
			o.OutputDir = v.Value().(string)
		case "target":
			o.Target = v.Value().(target.Double)
		default:
			switch flag {
			case "banner":
				o.JS.Banner = v.Value().(string)
			case "bundle":
				o.JS.Bundle = v.Value().(js.BundleMode)
			case "declaration-dir":
				o.JS.DeclarationDir = v.Value().(string)
			case "format":
				o.JS.Format = v.Value().(js.ModuleFormat)
			default:
				panic("unhandled flag: " + flag)
			}
			// Check if a JavaScript flag was used when not targeting JavaScript
			if t := o.Target.Target; t != target.JavaScript {
				cli.Failure(fmt.Sprintf(
					"Can't use JavaScript flag '%s' with target '%s'",
					argparse.FormatFlag(flag), t,
				))
			}
		}
	}
}

const LongDescription = ``
