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
		Options: []*build.Options{{BuildFile: build.BuildFile{
			FileConfiguration: build.FileConfiguration{
				JS: &build.FileJSOptions{},
			},
		}}},
		Verbose: r.Flag("verbose") == true,
	}
	defer b.CloseAll()
	b.Println("Starting build...")
	manifest := cli.ResolveManifest(projDir)
	b.Printf("Manifest found at %s\n", manifest)
	projDir = filepath.Dir(manifest)

	ParseFlags(r, b.Options[0])
	fmt.Println(manifest)
}

func ParseFlags(r *command.Runner, o *build.Options) {
	for flag, v := range r.AllFlags() {
		if v == nil {
			continue
		}
		switch flag {
		case "config":
			continue // TODO
		case "watch":
			o.Watch = v.Value().(bool)
		case "output":
			o.Output = []string{v.Value().(string)}
		case "target":
			o.Target = v.Value().(target.Target)
		default:
			switch flag {
			case "declaration":
				o.JS.Declaration = v.Value().(bool)
			case "minify":
				o.JS.Minify = v.Value().(bool)
			case "sourcemap":
				o.JS.Sourcemap = v.Value().(bool)
			case "jsdoc":
				o.JS.JSDoc = v.Value().(bool)
			case "copy-node-modules":
				o.JS.CopyNodeModules = v.Value().(bool)
			case "banner":
				o.JS.Banner = v.Value().(string)
			case "bundle":
				o.JS.Bundle = v.Value().(js.BundleMode)
			case "declaration-path":
				o.JS.DeclarationPath = v.Value().(string)
			case "format":
				o.JS.Format = v.Value().(js.ModuleFormat)
			default:
				panic("unhandled flag: " + flag)
			}
			// Check if a JavaScript flag was used when not targeting JavaScript
			// TODO: wait until all flags are parsed
			if o.Target != target.JavaScript {
				cli.Failure(fmt.Sprintf(
					"Can't use JavaScript flag '%s' with target '%s'",
					argparse.FormatFlag(flag), o.Target,
				))
			}
		}
	}
}

const LongDescription = `Compiles Klar source files at the provided file or module paths. If none are provided, inputs defined in 'klar.build', the build configuration file, are used.

An input passed to 'klar build' can be a directory path, to compile a module or package; a file path, to compile an individual file; '-', to compile as an individual file read from standard input, or a name prefixed with '@' to resolve a package and compile it.

A 'klar.build' is used to customize the build process and how files are compiled. For more information on build settings, see [url]. Common build options are provided as flags to override klar.build options.

Currently, Klar files can be compiled to JavaScript.`
