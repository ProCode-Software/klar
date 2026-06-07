package build

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/klarerrs/jsonerrors"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/pkg/argparse"
	"github.com/ProCode-Software/klar/pkg/klon"
)

// Build executes the "klar build" command.
func Build(r *command.Runner) {
	inputArgs := r.Parser.VarArgByName("inputs")
	cwd, err := build.Cwd()
	if err != nil {
		cli.FailureError(err)
	}
	c := build.NewCompiler(build.ModeBuild, cwd)
	pc := build.NewProjectCompiler(c)

	// Logging
	jsonOutput := r.Flag("json-output").Bool()
	if err := build.SetLogger(c, r.Flag("verbose").Bool(), jsonOutput); err != nil {
		cli.FailureError(err)
	}
	defer func() {
		if err := c.CloseLogger(); err != nil {
			cli.Failure("Failed to write log file: ", err)
		}
	}()

	c.StartTime = time.Now() // Start timer at resolution process

	// --config flag
	var (
		configFlag      = r.Flag("config")
		forcedKlarBuild *klarbuild.File
		klarBuildMode   int
	)
	if configFlag.String() != "" {
		var warn []*klon.Error
		forcedKlarBuild, warn, err = klarbuild.Parse(configFlag.String())
		if err != nil {
			c.FailWithError(err)
		}
		c.PrintKlonWarnings(warn, configFlag.String())
		ParseFlags(r, forcedKlarBuild)
		klarBuildMode = 1
	} else if configFlag.Set {
		klarBuildMode = 2 // Provided, but empty
	}

	// Resolve command-line inputs
	pc.Inputs = make([]build.ProjectInput, 0, len(inputArgs))
	addInput := func(path string) {
		input, err := pc.ResolveInput(path, klarBuildMode, false)
		if err != nil {
			c.FailWithError(err)
		}
		if forcedKlarBuild != nil {
			// Use the klar.build config from the --config flag
			input.Config = forcedKlarBuild
		} else {
			// Apply command-line flags
			ParseFlags(r, input.Config)
		}
		pc.Inputs = append(pc.Inputs, *input)
	}
	for _, path := range inputArgs {
		if path == "" {
			continue
		}
		addInput(path)
	}
	if len(pc.Inputs) == 0 {
		// If no inputs were provided, compile the current *package*
		pkgPath, _ := module.PackageRoot(".")
		c.Info("Building current package", slog.String("package", pkgPath))
		addInput(pkgPath)
	}

	// TODO: error if --output is file and there are multiple inputs
	res, err := pc.Compile()
	switch {
	case len(res.Errors) > 0:
		if r.Flag("sound-on-error").Bool() {
			playErrorSound()
		}
		printErrors(res, c, jsonOutput, err)
		cli.Exit(1)
	case err != nil:
		if jsonOutput {
			printJSONErrors(res, err, false)
			cli.Exit(1)
		}
		switch err := err.(type) {
		case *build.InterfaceError:
			// For InterfaceErrors: print a prettier error
			c.PrintInterfaceOrKlonError(err)
			cli.Exit(1)
		case *build.FilesystemError:
			cli.FailureError(err)
		default:
			// Errors should be a struct such as InterfaceError or FilesystemError
			panic(fmt.Sprintf("error %T should be wrapped: %[1]v", err))
		}
	default:
		ansi.Fprintfln(
			os.Stderr,
			"<**><g>%c</g> Build <g!>succeeded</g!></**> in <c>%s</c>!",
			icons.Check, util.FormatDuration(res.Elapsed),
		)
	}
}

// printErrors prints the "Build failed" message to standard error with the
// compile errors from res. isMaxErrors is whether compilation stopped early
// due to too many errors. Errors are printed using b's errorPrinter.
func printErrors(res *build.Result, b *build.Compiler, jsonOutput bool, err error) {
	errs := res.Errors
	// Format error count
	var count strings.Builder
	count.WriteString(strconv.Itoa(len(errs)))
	// Check to see if there were too many errors
	isMaxErrors := build.IsMaxErrors(err)
	if isMaxErrors {
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
	ansi.Fprintfln(
		os.Stderr,
		"<**><r>%c</r> Build <r!>failed</r!> with <r!>%s</r!></**> in <c>%s</c>\n",
		icons.ThinXLarge, count.String(), util.FormatDuration(res.Elapsed),
	)
	// Report the errors
	b.PrintAllErrors(errs)
	if isMaxErrors {
		cli.Error("There are too many errors")
	}
}

func printJSONErrors(res *build.Result, err error, isMaxErrors bool) {
	if err := jsonerrors.WriteTo(os.Stdout, res, err, isMaxErrors); err != nil {
		cli.Error("Failed to write JSON errors: ", err)
	}
	os.Stdout.WriteString("\n")
}

// ParseFlags parses flags from r into o.
func ParseFlags(r *command.Runner, f *klarbuild.File) {
	var firstJSFlag string
	for flag, v := range r.Flags {
		if v == nil {
			continue
		}
		switch flag {
		case "config", "verbose":
			continue // Already handled
		case "sound-on-error":
		case "watch":
			f.Watch = v.Bool()
		case "output":
			f.Output = []string{v.String()}
		case "target":
			f.Target = v.EnumValue().(target.Target)
		case "declaration":
			f.JS.Declaration = v.Bool()
		case "minify":
			f.JS.Minify = v.Bool()
		case "inline-sourcemap":
			if v.Bool() {
				f.JS.Sourcemap = klarbuild.SourceMapInline
			}
		case "sourcemap":
			if v.Bool() {
				f.JS.Sourcemap = klarbuild.SourceMapEnabled
			} else {
				f.JS.Sourcemap = klarbuild.SourceMapDisabled
			}
		case "jsdoc":
			f.JS.JSDoc = v.Bool()
		case "copy-node-modules":
			f.JS.CopyNodeModules = v.Bool()
		case "banner":
			f.JS.Banner = v.String()
		case "bundle":
			f.JS.Bundle = v.EnumValue().(klarbuild.BundleMode)
		case "declaration-path":
			f.JS.DeclarationPath = v.String()
		default:
			panic("unhandled flag: " + flag)
		}
		// Check if a JavaScript flag was used when not targeting JavaScript
		if f.JS != nil && !f.Target.IsJavaScript() && slices.Contains(jsFlags, flag) &&
			firstJSFlag == "" {
			firstJSFlag = flag
		}
	}
	if firstJSFlag != "" {
		cli.Failure(fmt.Sprintf(
			"Can't use JavaScript flag '%s' with target '%s'",
			argparse.FormatFlag(firstJSFlag), f.Target,
		))
	}
}

func playErrorSound() {
	// TODO: use a different path
	home, err := os.UserHomeDir()
	if err != nil {
		cli.Failure("Failed to get home directory: ", err)
	}
	soundPath := filepath.Join(home, "Downloads/fahh.mp3")

	// TODO: make this cross-platform
	cmd := exec.Command("pw-play", soundPath)
	if err := cmd.Start(); err != nil {
		cli.Failure("Failed to play error sound: ", err)
	}
}

const LongDescription = `Compiles Klar source files at the provided file or module paths. If none are provided, inputs defined in 'klar.build', the build configuration file, are used.

An input passed to 'klar build' can be a directory path, to compile a module or package; a file path, to compile an individual file; '-', to read from standard input and compile it as an individual file; or a name prefixed with '@' to resolve a module by its name and compile it.

A 'klar.build' is used to customize the build process and how files are compiled. For more information on build settings, run 'klar help klar.build'.
For each input, its closest 'klar.build' file is used to configure the build. The '--config' flag can be used to override the configuration for all inputs. Common build options are provided as flags to override klar.build options.

Currently, Klar files can be compiled to JavaScript.`
