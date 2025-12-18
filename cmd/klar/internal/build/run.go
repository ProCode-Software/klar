package build

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/argparse"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
)

// Build executes the "klar build" command.
func Build(r *command.Runner) {
	inputArgs := r.Parser.VarArgByName("inputs")
	b := build.NewCompiler(build.ModeBuild)
	if err := cmp.Or(
		b.UseStdOpener(),
		b.SetLogger(r.BoolFlag("verbose")),
	); err != nil {
		cli.Failure(err.Error())
	}
	delete(r.AllFlags(), "verbose") // Don't reparse it

	defer b.CloseAll()
	// Resolve all inputs
	if len(inputArgs) > 0 {
		b.Logf("Resolving inputs: %v\n", inputArgs)
	}
	b.StartTime = time.Now()
	inps, err := build.ResolveInputs(inputArgs)
	// Build the nearest *package* if no path provided
	if err == nil && len(inps) == 0 {
		pkgPath, _, err := module.PackageRoot(".")
		if err != nil {
			cli.ErrNoManifest(pkgPath)
		}
		b.Log("Resolving inputs at current package:", pkgPath)
		//nolint:ineffassign // False positive
		inps, err = build.ResolveInputs([]string{pkgPath})
	}
	if err != nil {
		// Show a better error for file not found
		if fsErr, ok := err.(*build.FilesystemError); ok && fsErr.IsNotExist() {
			cli.ErrNotFound(fsErr.Path, "")
		} else if ierr, ok := err.(*build.InterfaceError); ok {
			build.PrintInterfaceErr(ierr)
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
	res, err := b.Compile()
	// For InterfaceErrors: print a prettier error
	ierr, isInterfaceErr := err.(*build.InterfaceError)
	switch {
	case res.EarlyExit, len(res.Errors) > 0:
		printErrors(res, res.EarlyExit, b)
	case isInterfaceErr:
		build.PrintInterfaceErr(ierr)
	case err != nil:
		// Errors should be a struct such as InterfaceError or FilesystemError
		panic(fmt.Sprintf("error %T should be wrapped: %[1]v", err))
	default:
		ansi.Fprintfln(os.Stderr,
			"<**><g>%c</g> Build <g!>succeeded</g!></**> in <c>%s</c>!",
			icons.Check, cli.FormatDuration(res.Elapsed),
		)
	}
}

// printErrors prints the "Build failed" message to standard error with the
// compile errors from res. isMaxErrors is whether compilation stopped early
// due to too many errors. Errors are printed using b's errorPrinter.
func printErrors(res *build.BuildResult, isMaxErrors bool, b *build.Compiler) {
	errs := res.Errors
	// Format error count
	var count strings.Builder
	count.WriteString(strconv.Itoa(len(errs)))
	if isMaxErrors {
		count.WriteByte('+')
	}
	count.WriteString(" error")
	if len(errs) != 1 {
		count.WriteByte('s')
	}
	// Show "build failed" message
	ansi.Fprintfln(os.Stderr,
		"<**><r>%c</r> Build <r!>failed</r!> with <r!>%s</r!></**> in <c>%s</c>",
		icons.ThinXLarge, count.String(), cli.FormatDuration(res.Elapsed),
	)
	// Report the errors
	for _, err := range errs {
		b.PrintError(err)
	}
	if isMaxErrors {
		cli.Error("There are too many errors")
	}
}

// ParseFlags parses flags from r into o.
func ParseFlags(r *command.Runner, o *build.Options) {
	for flag, v := range r.AllFlags() {
		if v == nil {
			continue
		}
		switch flag {
		case "config", "verbose":
			continue // Already handled
		case "watch":
			o.Watch = v.Value().(bool)
		case "output":
			o.Output = []string{v.Value().(string)}
		case "target":
			o.Target = v.Value().(target.Target)
		default:
			// The rest are all JS flags
			if o.JS == nil {
				o.JS = &klarbuild.JSOptions{}
			}
			// Check if a JavaScript flag was used when not targeting JavaScript
			// TODO: wait until all flags are parsed. Flags aren't in order.
			if o.Target != target.JavaScript {
				cli.Failure(fmt.Sprintf(
					"Can't use JavaScript flag '%s' with target '%s'",
					argparse.FormatFlag(flag), o.Target,
				))
			}
			switch flag {
			case "declaration":
				o.JS.Declaration = v.Value().(bool)
			case "minify":
				o.JS.Minify = v.Value().(bool)
			case "inline-sourcemap":
				if v.Value().(bool) {
					o.JS.Sourcemap = klarbuild.SourceMapInline
				}
			case "sourcemap":
				if v.Value().(bool) {
					o.JS.Sourcemap = klarbuild.SourceMapEnabled
				}
			case "jsdoc":
				o.JS.JSDoc = v.Value().(bool)
			case "copy-node-modules":
				o.JS.CopyNodeModules = v.Value().(bool)
			case "banner":
				o.JS.Banner = v.Value().(string)
			case "bundle":
				o.JS.Bundle = v.Value().(klarbuild.BundleMode)
			case "declaration-path":
				o.JS.DeclarationPath = v.Value().(string)
			case "format":
				o.JS.Format = v.Value().(klarbuild.ModuleFormat)
			default:
				panic("unhandled flag: " + flag)
			}
		}
	}
}

const LongDescription = `Compiles Klar source files at the provided file or module paths. If none are provided, inputs defined in 'klar.build', the build configuration file, are used.

An input passed to 'klar build' can be a directory path, to compile a module or package; a file path, to compile an individual file; '-', to read from standard input and compile it as an individual file; or a name prefixed with '@' to resolve a module by its name and compile it.

A 'klar.build' is used to customize the build process and how files are compiled. For more information on build settings, see [url].
For each input, its closest 'klar.build' file is used to configure the build. The '--config' flag can be used to override the configuration for all inputs. Common build options are provided as flags to override klar.build options.

Currently, Klar files can be compiled to JavaScript.`
