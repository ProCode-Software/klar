package build

import (
	"fmt"
	"log/slog"
	"maps"
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
	// ==========
	jsonOutput := r.Flag("json-output").Bool()
	if err := build.SetLogger(c, r.Flag("verbose").Bool(), jsonOutput); err != nil {
		cli.FailureError(err)
	}
	defer func() {
		if err := c.CloseLogger(); err != nil {
			cli.Failure("Failed to write log file:", err)
		}
	}()

	c.StartTime = time.Now() // Start timer at resolution process
	if !jsonOutput {
		c.Progress = NewBuildStatus(cwd)
	}

	// --config flag
	// =========
	var (
		configFlag      = r.Flag("config")
		forcedKlarBuild *klarbuild.File
		klarBuildMode   int
	)
	if configFlag.String() != "" {
		configPath := configFlag.String()
		var warn []*klon.Error
		if forcedKlarBuild, warn, err = klarbuild.Parse(configPath); err != nil {
			c.FailWithError(err)
		}
		c.PrintKlonWarnings(warn, configPath)
		ParseFlags(r, forcedKlarBuild)
		klarBuildMode = 1
	} else if configFlag.Set {
		klarBuildMode = 2 // Provided, but empty. Use default config
	}
	// If unset, a klar.build will be searched for

	// Resolve command-line inputs
	// =========
	pc.Inputs = make([]*build.Input, 0, len(inputArgs))
	addInput := func(path string) {
		input, err := pc.ResolveInput(path, klarBuildMode, false)
		switch {
		case err != nil:
			c.FailWithError(err)
		case forcedKlarBuild != nil:
			// Use the klar.build config from the --config flag
			input.KlarBuild = forcedKlarBuild
		default:
			// Apply command-line flags
			ParseFlags(r, input.KlarBuild)
			if targetFlag := r.Flag("target"); targetFlag.Set {
				input.Targets = []target.Target{targetFlag.EnumValue().(target.Target)}
			}
		}
		pc.Inputs = append(pc.Inputs, input)
	}
	for i, path := range inputArgs {
		if path == "" {
			continue
		}
		c.Progress.ResolvingInput(path, i+1, len(inputArgs))
		addInput(path)
	}
	if len(pc.Inputs) == 0 {
		// If no inputs were provided, compile the current *package*
		pkgPath, _ := module.PackageRoot(".")
		c.Info("Building current package", slog.String("package", pkgPath))
		addInput(pkgPath)
	}

	// Resolve lockfiles and download dependencies for each input
	// ========
	if err := pc.DownloadDeps(); err != nil {
		c.FailWithError(err)
	}

	// Compile!
	// ==========
	// TODO: error if --output is file and there are multiple inputs
	res, err := pc.Compile()

	// Print error/success messages
	// ===========
	switch {
	case err != nil:
		if jsonOutput {
			printJSONErrors(res, err)
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
	case len(res.Errors) > 0:
		if r.Flag("sound-on-error").Bool() {
			playErrorSound()
		}
		printErrors(res, c, jsonOutput)
		cli.Exit(1)
	default:
		if jsonOutput {
			printJSONErrors(res, err)
			break
		}
		ansi.Fprintfln(
			os.Stderr,
			"%s<**><g>%c</g> Build <g!>succeeded</g!></**> in <c>%s</c>!",
			ansi.ClearLine, icons.Check, util.FormatDuration(res.Elapsed),
		)
	}
}

// printErrors prints the "Build failed" message to standard error with the
// compile errors from res. isMaxErrors is whether compilation stopped early
// due to too many errors. Errors are printed using b's errorPrinter.
func printErrors(res *build.Result, c *build.Compiler, jsonOutput bool) {
	errs := res.Errors
	// Format error count
	var count strings.Builder
	count.WriteString(strconv.Itoa(len(errs)))
	// Check to see if there were too many errors
	if res.IsMaxErrors {
		count.WriteByte('+')
	}
	count.WriteString(" error")
	if len(errs) != 1 {
		count.WriteByte('s')
	}
	// Print JSON errors if jsonOutput is true
	if jsonOutput {
		printJSONErrors(res, nil)
		return
	}
	// Show "build failed" message
	ansi.Fprintfln(
		os.Stderr,
		"%s<**><r>%c</r> Build <r!>failed</r!> with <r!>%s</r!></**> in <c>%s</c>\n",
		ansi.ClearLine, icons.ThinXLarge, count.String(), util.FormatDuration(res.Elapsed),
	)
	// Report the errors
	c.PrintAllErrors(errs)
	if res.IsMaxErrors {
		cli.Error("There are too many errors")
	}
}

func printJSONErrors(res *build.Result, err error) {
	isMaxErrors := res != nil && res.IsMaxErrors
	if err := jsonerrors.WriteTo(os.Stdout, res, err, isMaxErrors); err != nil {
		cli.Error("Failed to write JSON errors:", err)
	}
	os.Stdout.WriteString("\n")
}

// ParseFlags parses flags from r into o.
func ParseFlags(r *command.Runner, f *klarbuild.File) {
	for _, setting := range klarBuildFlags {
		flag, ok := r.Flags[setting]
		if !ok || !flag.Set {
			continue
		}
		switch setting {
		case "watch":
			f.Watch = flag.Value.(bool)
		case "output":
			f.Output = flag.Value.([]string)
		case "target":
			f.Target = flag.Value.(target.Target)
		default:
			panic("unhandled flag: " + setting)
		}
	}
	for _, setting := range jsFlags {
		v, ok := r.Flags[setting]
		if !ok || !v.Set {
			continue
		}
		// Check if a JavaScript flag was used when not targeting JavaScript
		if !f.Target.IsJavaScript() {
			// Get the first JS flag the user provided
			firstJSFlag := slices.SortedFunc(maps.Keys(r.Flags), func(a, b string) int {
				return r.Flags[a].Index - r.Flags[b].Index
			})[0]
			cli.Failure(fmt.Sprintf(
				"Can't use JavaScript flag '%s' with target '%s'",
				argparse.FormatFlag(firstJSFlag), f.Target,
			))
		}
		switch setting {
		case "declaration":
			f.JS.Declaration = v.Value.(bool)
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
			panic("unhandled flag: " + setting)
		}
	}
}

func playErrorSound() {
	// TODO: use a different path
	home, err := os.UserHomeDir()
	if err != nil {
		cli.Failure("Failed to get home directory:", err)
	}
	soundPath := filepath.Join(home, "Downloads/fahh.mp3")

	// TODO: make this cross-platform
	cmd := exec.Command("pw-play", soundPath)
	if err := cmd.Start(); err != nil {
		cli.Failure("Failed to play error sound:", err)
	}
}

const LongDescription = `Compiles Klar source files at the provided file or module paths. If none are provided, inputs defined in 'klar.build', the build configuration file, are used.

An input passed to 'klar build' can be a directory path, to compile a module or package; a file path, to compile an individual file; '-', to read from standard input and compile it as an individual file; or a name prefixed with '@' to resolve a module by its name and compile it.

A 'klar.build' is used to customize the build process and how files are compiled. For more information on build settings, run 'klar help klar.build'.
For each input, its closest 'klar.build' file is used to configure the build. The '--config' flag can be used to override the configuration for all inputs. If the '--config' flags is provided, but empty, the default settings are used without looking for 'klar.build' files. Common build options are provided as flags to override klar.build options.

Currently, Klar files can be compiled to JavaScript.`
