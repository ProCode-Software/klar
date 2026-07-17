package build

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
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

	// 1. Logging
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

	// 2. --config flag
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
		ParseFlags(r, &build.Input{KlarBuild: forcedKlarBuild})
		klarBuildMode = 1
	} else if configFlag.Set {
		klarBuildMode = 2 // Provided, but empty. Use default config
	}
	// If unset, a klar.build will be searched for

	// 3. Resolve command-line inputs
	pc.Inputs = make([]*build.Input, 0, len(inputArgs))
	addInput := func(path string) {
		input, err := pc.ResolveInput(path, klarBuildMode)
		switch {
		case err != nil:
			if err, ok := err.(*build.FilesystemError); ok && err.IsNotExist() {
				cli.ErrNotFound(path, "input")
			}
			c.FailWithError(err)
		case forcedKlarBuild != nil:
			// Use the klar.build config from the --config flag
			input.KlarBuild = forcedKlarBuild
		default:
			// Apply command-line flags
			ParseFlags(r, input)
		}
		pc.Inputs = append(pc.Inputs, input)

		// Hide progress if stdin is an input
		if input.Kind == build.KindStdin {
			c.Progress = build.HiddenProgress{}
			fmt.Print(ansi.ClearLine) // Clear the line created by ResolvingInput
		}
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
		pkgPath, _ := module.PackageRoot(c.WorkDir)
		c.Info("Building current package", slog.String("package", pkgPath))
		addInput(pkgPath)
	}

	// 4. Resolve lockfiles and download dependencies for each input
	if err := pc.DownloadDeps(); err != nil {
		c.FailWithError(err)
	}

	// 5. Compile!
	// TODO: error if --output is file and there are multiple inputs
	res, err := pc.Compile()

	// 6. Print error/success messages

	// If we're showing the compiler's progress, clear the line before showing errors
	if !c.ProgressHidden() {
		fmt.Print(ansi.ClearLine)
	}
	switch {
	case jsonOutput:
		isMaxErrors := res != nil && res.IsMaxErrors
		if err := jsonerrors.WriteTo(os.Stdout, res, err, isMaxErrors); err != nil {
			cli.Error("Failed to write JSON errors:", err)
		}
		fmt.Println()
		if err != nil || len(res.Errors) > 0 {
			cli.Exit(1)
		}
	case err != nil:
		// Critical error
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
		// Successes, errors, and/or warnings
		showResult(res, c)
	}
}

func showResult(res *build.Result, c *build.Compiler) {
	var (
		warnCount, errCount = len(res.Warnings), len(res.Errors)
		icon, format        string
	)
	formatIcon := func(icon rune, color byte) string {
		return fmt.Sprintf("<%c>%c</%[1]c>", color, icon)
	}
	formatCount := func(n int, kind string) string {
		switch {
		case n == 1:
			return fmt.Sprintf("%d %s", n, kind)
		case kind == "error" && res.IsMaxErrors:
			return fmt.Sprintf("%d+ %ss", build.MaxErrors, kind)
		}
		return fmt.Sprintf("%d %ss", n, kind)
	}
	switch {
	case errCount == 0 && warnCount == 0:
		// Succeeded
		icon, format = formatIcon(icons.Check, 'g'), "<g!>succeeded</g!>"
	case errCount == 0 && warnCount > 0:
		// Succeeded with warnings
		icon, format = formatIcon(icons.Warning, 'y'),
			"<y!>succeeded</y!> with <g!>"+formatCount(warnCount, "warning")+"</g!>"
	case errCount > 0 && warnCount == 0:
		// Failed
		icon = formatIcon(icons.ThinXLarge, 'r')
		format = "<r!>failed</r!> with <r!>" + formatCount(errCount, "error") + "</r!>"
	case errCount > 0 && warnCount > 0:
		// Failed with errors and warnings
		icon = formatIcon(icons.ThinXLarge, 'r')
		format = "<r!>failed</r!> with <r!>" + formatCount(errCount, "error") +
			"</r!> and <y!>" + formatCount(warnCount, "warning") + "</y!>"
	default:
		panic(fmt.Sprintf("unreachable: %d errors and %d warnings", errCount, warnCount))
	}
	dur := util.FormatDuration(res.Elapsed)
	fmt.Fprintln(os.Stderr, ansi.Colorize(
		"<**>"+icon+" Build "+format+"</**> in <c>"+dur+"</c>",
	))

	if errCount > 0 || warnCount > 0 {
		fmt.Fprintln(os.Stderr)
	}
	c.PrintAllErrors(res.Errors)
	c.PrintAllErrors(res.Warnings)
	if res.IsMaxErrors {
		fmt.Fprintln(os.Stderr, ansi.BoldBrightRed(
			ansi.Underline("\nBuild stopped early due to too many errors"),
		))
	}
	if errCount > 0 {
		cli.Exit(1)
	}
}

// ParseFlags parses flags from r into o.
func ParseFlags(r *command.Runner, i *build.Input) {
	for _, setting := range klarBuildFlags {
		flag, ok := r.Flags[setting]
		if !ok || !flag.Set {
			continue
		}
		if i.KlarBuild == nil && setting != "target" {
			i.KlarBuild = &klarbuild.File{}
		}
		f := i.KlarBuild
		switch setting {
		case "watch":
			f.Watch = flag.Value.(bool)
		case "output":
			f.Output = flag.Value.([]string)
		case "target":
			i.Targets = []target.Target{flag.EnumValue().(target.Target)}
			if f != nil {
				f.Target = i.Targets[0]
			}
		default:
			panic("unhandled flag: " + setting)
		}
	}
	for _, setting := range jsFlags {
		v, ok := r.Flags[setting]
		if !ok || !v.Set {
			continue
		}
		if i.KlarBuild == nil {
			i.KlarBuild = &klarbuild.File{}
		}
		f := i.KlarBuild
		// Check if a JavaScript flag was used when not targeting JavaScript
		isJS := func(t target.Target) bool { return t.IsJavaScript() }
		if !slices.ContainsFunc(i.Targets, isJS) {
			// Get the first JS flag the user provided
			firstJSFlag := slices.SortedFunc(maps.Keys(r.Flags), func(a, b string) int {
				return r.Flags[a].Index - r.Flags[b].Index
			})[0]
			cli.Failure(fmt.Sprintf(
				"Can't use JavaScript flag '--%s' with target '%s'",
				firstJSFlag, f.Target,
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

const LongDescription = `Compiles Klar source files at the provided file or module paths. If none are provided, inputs defined in 'klar.build', the build configuration file, are used.

An input passed to 'klar build' can be a directory path, to compile a module or package; a file path, to compile an individual file; '-', to read from standard input and compile it as an individual file; or a name prefixed with '@' to resolve a module by its name and compile it.

A 'klar.build' is used to customize the build process and how files are compiled. For more information on build settings, run 'klar help klar.build'.
For each input, its closest 'klar.build' file is used to configure the build. The '--config' flag can be used to override the configuration for all inputs. If the '--config' flags is provided, but empty, the default settings are used without looking for 'klar.build' files. Common build options are provided as flags to override klar.build options.

Currently, Klar files can be compiled to JavaScript.`
