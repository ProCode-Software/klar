package build

import (
	"fmt"
	"log/slog"
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
	"github.com/ProCode-Software/klar/internal/errors/jsonerrors"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
)

// Build executes the "klar build" command.
func Build(r *command.Runner) {
	var (
		inputArgs = r.Parser.VarArgByName("inputs")
		b         = build.NewCompiler(build.ModeBuild)
		err       = b.UseStdOpener()
	)
	if err != nil {
		cli.FailureError(err)
	}
	jsonOutput := r.BoolFlag("json-output")
	if err := build.SetLogger(b, r.BoolFlag("verbose"), jsonOutput); err != nil {
		cli.FailureError(err)
	}
	defer func() {
		if err := b.CloseLogger(); err != nil {
			cli.Failure("Failed to write log file: ", err)
		}
	}()
	delete(r.AllFlags(), "verbose") // Avoid reparsing flags in [ParseFlags]
	delete(r.AllFlags(), "json-output")

	// Resolve all inputs if provided
	if len(inputArgs) > 0 {
		b.LogInfo("Resolving inputs", slog.Any("inputs", inputArgs))
	}
	b.StartTime = time.Now() // Start timer at resolution process
	var configPath string    // Config path if resolved from cwd or --config flag
	inps, err := build.ResolveInputs(inputArgs)
	if err == nil && len(inps) == 0 {
		// Try reading from the cwd's klar.build if no inputs provided
		if _, err := os.Stat("klar.build"); err == nil {
			configPath = "klar.build"
			b.LogInfo("klar.build found in current directory")
		} else {
			// Build the nearest *package* if no path provided
			pkgPath, _, err := module.PackageRoot(".")
			if err != nil {
				cli.ErrNoManifest(pkgPath)
			}
			b.LogInfo("Resolving inputs at current package", slog.String("package", pkgPath))
			//nolint:ineffassign // False positive
			inps, err = build.ResolveInputs([]string{pkgPath})
		}
	}
	if err != nil {
		// Show a better error for file not found
		if fe, ok := err.(*build.FilesystemError); ok && fe.IsNotExist() {
			cli.ErrNotFound(fe.Path, "")
		} else if ie, ok := err.(*build.InterfaceError); ok {
			build.PrintInterfaceErr(ie)
			cli.Exit(1)
		}
		cli.Failure(err.Error())
	}
	// Force a config path if --config flag was passed
	var configFlag *build.Options
	if conf := r.StringFlag("config"); conf != "klar.build" {
		configPath = conf
		cfs, err := build.ReadKlarBuild(conf)
		if err != nil {
			build.PrintInterfaceErr(err.(*build.InterfaceError))
		} else if len(cfs) == 0 {
			// Make sure the --config has options in it
			cli.Failuref("The configuration from '%s' has no options in it", "", conf)
		}
		b.LogInfo("Using --config flag:", slog.String("path", conf))
		configFlag = cfs[0]
		delete(r.AllFlags(), "config")
	}

	// Read options from klar.build
	if len(inps) == 0 && configPath != "" {
		if b.Options, err = build.ReadKlarBuild(configPath); err != nil {
			build.PrintInterfaceErr(err.(*build.InterfaceError))
		}
	} else {
		b.Options = make([]*build.Options, 0, len(inps))
	}
	// Apply options for each input
	for _, inp := range inps {
		var opt *build.Options
		if configPath != "" {
			// Use --config flag
			opt = configFlag
			inp.KlarBuild = configPath
		} else {
			// Use the Input's klar.build
			opts, err := build.ReadKlarBuild(inp.KlarBuild)
			switch {
			case err != nil:
				build.PrintInterfaceErr(err.(*build.InterfaceError))
			case len(opts) == 0:
				opt = build.ParseKlarBuild(klarbuild.Default(), nil)
			default:
				opt = opts[0]
			}
		}
		ParseFlags(r, opt)
		opt.Inputs = []build.Input{inp}
		b.Options = append(b.Options, opt)
	}
	// TODO: error if --output is file and there are multiple inputs
	res, err := b.Compile()
	switch {
	case len(res.Errors) > 0:
		printErrors(res, b, jsonOutput, err)
		cli.Exit(1)
	case err != nil:
		if jsonOutput {
			printJSONErrors(res, err, false)
			cli.Exit(1)
		}
		switch err := err.(type) {
		case *build.InterfaceError:
			// For InterfaceErrors: print a prettier error
			build.PrintInterfaceErr(err)
			cli.Exit(1)
		case *build.FilesystemError:
			cli.Failure(err.Error())
		default:
			// Errors should be a struct such as InterfaceError or FilesystemError
			panic(fmt.Sprintf("error %T should be wrapped: %[1]v", err))
		}
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
func printErrors(res *build.BuildResult, b *build.Compiler, jsonOutput bool, err error) {
	errs := res.Errors
	// Format error count
	var count strings.Builder
	count.WriteString(strconv.Itoa(len(errs)))
	// Check to see if there were too many errors
	var isMaxErrors bool
	if err, ok := err.(*build.InterfaceError); ok && err.Code == build.ErrTooManyErrors {
		isMaxErrors = true
		count.WriteByte('+')
	}
	count.WriteString(" error")
	if len(errs) != 1 {
		count.WriteByte('s')
	}
	// Print JSON errors if jsonOutput is true
	if jsonOutput {
		printJSONErrors(res, err, isMaxErrors)
		return
	}
	// Show "build failed" message
	ansi.Fprintfln(os.Stderr,
		"<**><r>%c</r> Build <r!>failed</r!> with <r!>%s</r!></**> in <c>%s</c>",
		icons.ThinXLarge, count.String(), cli.FormatDuration(res.Elapsed),
	)
	// Report the errors
	for i, err := range errs {
		if i > 0 {
			fmt.Fprintln(os.Stderr)
		}
		b.PrintError(err)
	}
	if isMaxErrors {
		cli.Error("There are too many errors")
	}
}

func printJSONErrors(res *build.BuildResult, err error, isMaxErrors bool) {
	if err := jsonerrors.WriteTo(os.Stdout, res, err, isMaxErrors); err != nil {
		cli.Error("Failed to write JSON errors: ", err)
	}
	os.Stdout.WriteString("\n")
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
