package build

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/argparse"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
)

func Build(r *command.Runner) {
	inputArgs := r.Parser.VarArgByName("inputs")
	b := &build.Compiler{Mode: build.ModeBuild}
	if r.BoolFlag("verbose") {
		b.Verbose = true
		delete(r.AllFlags(), "verbose")
	}
	b.InitLogger()
	defer b.CloseAll()

	inps, err := build.ResolveInputs(inputArgs) // Resolve all inputs
	// Build the nearest *package* if no path provided
	if err == nil && len(inps) == 0 {
		pkgPath, _, err := module.PackageRoot(".")
		if err != nil {
			cli.ErrNoManifest(pkgPath)
		}
		inps, err = build.ResolveInputs([]string{pkgPath})
	}
	if err != nil {
		// Show a better error for file not found
		if err, ok := err.(*build.FilesystemError); ok && err.IsNotExist() {
			cli.ErrNotFound(err.Path, "")
		}
		cli.Failure(err.Error())
	}
	// Force a config path if --config flag was passed
	var forceConfig string
	if conf := r.StringFlag("config"); conf != "" {
		forceConfig = conf
		delete(r.AllFlags(), "config")
	}
	// Apply options for each input
	b.Options = make([]*build.Options, 0, len(inps))
	for _, inp := range inps {
		if forceConfig != "" {
			inp.KlarBuild = forceConfig
		}
		opt := &build.Options{} // TODO: parse klar.build here
		ParseFlags(r, opt)
		opt.Inputs = []build.Input{inp}
		b.Options = append(b.Options, opt)
	}
	// TODO: error if --output is file and there are multiple inputs
	
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
			// The rest are all JS flags
			if o.JS == nil {
				o.JS = &build.FileJSOptions{}
			}
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

An input passed to 'klar build' can be a directory path, to compile a module or package; a file path, to compile an individual file; '-', to read from standard input and compile it as an individual file; or a name prefixed with '@' to resolve a module by its name and compile it.

A 'klar.build' is used to customize the build process and how files are compiled. For more information on build settings, see [url].
For each input, its closest 'klar.build' file is used to configure the build. The '--config' flag can be used to override the configuration for all inputs. Common build options are provided as flags to override klar.build options.

Currently, Klar files can be compiled to JavaScript.`
