package build

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/build/js"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/argparse"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
)

func Build(r *command.Runner) {
	inputArgs := r.Parser.VarArgByName("inputs")
	b := &build.Compiler{
		Mode:    build.ModeBuild,
		Verbose: r.BoolFlag("verbose"),
	}
	delete(r.AllFlags(), "verbose") // Don't reparse it
	b.StartTime = time.Now()
	b.InitLogger()
	defer b.CloseAll()
	// Resolve all inputs
	if len(inputArgs) > 0 {
		b.Logf("Resolving inputs: %v\n", inputArgs)
	}
	inps, err := build.ResolveInputs(inputArgs)
	// Build the nearest *package* if no path provided
	if err == nil && len(inps) == 0 {
		pkgPath, _, err := module.PackageRoot(".")
		if err != nil {
			cli.ErrNoManifest(pkgPath)
		}
		b.Log("Resolving inputs at current package:", pkgPath)
		inps, err = build.ResolveInputs([]string{pkgPath})
	}
	if err != nil {
		// Show a better error for file not found
		if err, ok := err.(*build.FilesystemError); ok && err.IsNotExist() {
			cli.ErrNotFound(err.Path, "")
		}
		if err, ok := err.(*build.InterfaceError); ok {
			printInterfaceErr(err)
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
	intfErr, isIntfErr := err.(*build.InterfaceError)
	// For a TooManyErrors InterfaceError, still print the other error messages
	isMaxErrors := isIntfErr && intfErr.Code == build.ErrTooManyErrors
	switch {
	case isMaxErrors, len(res.Errors) > 0:
		printErrors(res, isMaxErrors, b)
	case isIntfErr:
		printInterfaceErr(intfErr)
	case err != nil:
		cli.Failure("", err) // TODO: categorize errors (struct)
	default:
		fmt.Fprintln(os.Stderr, ansi.BoldGreen(string(icons.Check)),
			ansi.Bold("Build"), ansi.BoldBrightGreen("succeeded"),
			"in", ansi.Cyan(cli.FormatDuration(res.Elapsed))+"!",
		)
	}
}

func printInterfaceErr(err *build.InterfaceError) {
	main, detail := err.PrettyError()
	cli.Failure(
		ansi.Bold(strings.Join(main, ansi.Partial(ansi.CodeBold))),
		detail,
	)
}

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
	fmt.Fprintln(os.Stderr, ansi.BoldRed(string(icons.ThinXLarge)),
		ansi.Bold("Build"), ansi.BoldBrightRed("failed"), ansi.Bold("with"),
		ansi.BoldBrightRed(count.String()),
		"in", ansi.Cyan(cli.FormatDuration(res.Elapsed))+ansi.BoldDim(":"),
	)
	// Report the errors
	for _, err := range errs {
		b.ErrorPrinter.PrintError(err)
	}
	if isMaxErrors {
		cli.Error("There are too many errors")
	}
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
