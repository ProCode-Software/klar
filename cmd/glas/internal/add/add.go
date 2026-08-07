package add

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ProCode-Software/klar/cmd/glas/internal/monorepo"
	"github.com/ProCode-Software/klar/cmd/glas/internal/spinner"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/prompt"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/internal/util/manifest"
	"github.com/ProCode-Software/klar/internal/version"
	"github.com/ProCode-Software/klar/pkg/argparse"
	"golang.org/x/term"
)

type Package interface {
	Name() string
	Source() pkgSource
	Specifier() version.Specifier
	setNameAndSpec(name string, spec version.Specifier)
	ResolvedVersion() string
	Info(*installContext) *pkgInfo
	Install(*installContext)
}

type pkg struct {
	name        string
	versionSpec version.Specifier
	src         pkgSource
}

type pkgSource int

const (
	gitSource pkgSource = iota
	npmSource
	localSource
	workspaceSource
)

type installContext struct {
	targetPkgs []*module.PackageInfo
	*slog.Logger
	*argparse.Parser
	debug bool
}

func Run(p *argparse.Parser) {
	pkgs := p.VarArgByName("packages")
	if len(pkgs) == 0 {
		// We mainly didn't use angle brackets in [Flags] so we could display this message
		cli.Error("You didn't provide any packages to install!")
		cli.Hint(
			"To install all of the project's dependencies, run " +
				ansi.BrightCyan("glas install") + " instead.",
		)
		cli.Exit(2)
	}
	ic := &installContext{Parser: p}

	// Setup verbose logging
	logger, err := util.SetLogger(p.Flag("verbose").Bool(), false)
	if err != nil {
		cli.FailureError(err)
	}
	ic.Logger = logger
	if ic.Logger.Enabled(context.TODO(), slog.LevelDebug) {
		ic.debug = true
	}
	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	// 1. Get the packages the deps will be installed for
	projInfo := manifest.GetPackageInfo("")
	switch {
	case p.Flag("global").Bool():
		if p.Flag("for").Set {
			cli.Failure("Can't use '--for' flag when installing globally")
		}
		cli.Failure("Installing global commands isn't implemented yet")
	case p.Flag("for").Set:
		patterns := p.Flag("for").Value.([]string)
		subpkgs := monorepo.ParseForFlag(projInfo.ProjectDir, patterns)
		got := make(map[string]struct{}) // For deduping
		for _, pkgName := range subpkgs {
			if _, ok := got[pkgName]; ok {
				continue
			}
			ic.parseSubpackage(pkgName, projInfo)
			got[pkgName] = struct{}{}
		}
	case manifest.IsMonorepoRoot(projInfo):
		if p.Flag("yes").Bool() || p.Flag("project").Bool() || !isInteractive {
			// Assume they're installing for the whole project
			ic.targetPkgs = []*module.PackageInfo{projInfo}
			break
		}
		ic.promptSpecificPackages(projInfo)
	default:
		ic.targetPkgs = []*module.PackageInfo{projInfo}
	}

	type pkgPair struct {
		pkg  Package
		info *pkgInfo
	}
	parsedPkgs := make([]pkgPair, len(pkgs))

	// 2. Now that we have the target packages loaded, we can finally install
	// each package. Do this in parallel because of NPM requests, and cloning
	// Git repos (shallow until confirmed).
	var wg sync.WaitGroup
	for i, inp := range pkgs {
		wg.Go(func() {
			defer cli.HandleSignalExit() // When [cli.Failure] is called
			pkg := ic.parseInput(inp)
			info := pkg.Info(ic)
			parsedPkgs[i] = pkgPair{pkg, info}
		})
	}

	// Loader while waiting for the info to be fetched
	done := make(chan struct{})
	go spinner.Circle(fmt.Sprintf(
		ansi.Magenta("Getting information for %s"),
		klarerrs.FormatCount(len(parsedPkgs), "package"),
	), done)
	wg.Wait()
	close(done)
	fmt.Print(ansi.ClearLine)

	rejected := make(map[int]struct{})
	// 3. Ask for confirmation
	for i, pair := range parsedPkgs {
		yes := p.Flag("yes").Bool()
		if pair.info == nil {
			continue
		}
		if len(parsedPkgs) > 1 {
			ansi.ColorPrintfln(
				ansi.CodeDim+ansi.CodeUnderline,
				"Package %d of %d", i+1, len(parsedPkgs),
			)
		}
		showInfo(pair.pkg, pair.info)
		if !yes && isInteractive {
			choice := prompt.ChooseLetter(
				ansi.BoldBrightYellow("Do you want to install?"),
				map[byte]string{'y': "Yes", 'n': "No", 'x': "No to all"}, 'y',
			)
			switch choice {
			case 'n':
				rejected[i] = struct{}{}
			case 'x':
				ansi.ColorPrintln(ansi.CodeBrightRed, "Cancelled installing all packages")
				cli.Exit(1)
			}
		}
		if i < len(parsedPkgs)-1 {
			fmt.Println()
		}
	}
	if len(rejected) == len(parsedPkgs) {
		ansi.ColorPrintln(ansi.CodeBrightRed, "Cancelled installing all packages")
		cli.Exit(1)
	}

	// 4. Install accepted packages
	for i, pkg := range parsedPkgs {
		if _, ok := rejected[i]; ok {
			continue
		}
		pkg.pkg.Install(ic)
	}
}

// Shown if 'glas add' is being run in the root of a monorepo
func (ic *installContext) promptSpecificPackages(projInfo *module.PackageInfo) {
	dirs, err := os.ReadDir(filepath.Join(projInfo.ProjectDir, module.PkgDir))
	if err != nil {
		cli.Failuref("Failed to list %s/pkg: %v", projInfo.ProjectDir, err)
	}
	const allMsg = "Entire project (all current & future subpackages)"
	selected := make(map[string]bool, len(dirs))
	selected[allMsg] = false
	for _, dir := range dirs {
		if dir.IsDir() {
			selected[dir.Name()] = false
		}
	}
	// If the user runs 'glas add' from the monorepo root, prompt
	// them which subpackages to install the package to
	prompt.Checkboxes(
		"Choose the packages to install dependencies to",
		"You're running 'glas add' in your project root. Choose the subpackages you want to add the dependency to.",
		selected, false,
	)

	if selected[allMsg] {
		ic.targetPkgs = []*module.PackageInfo{projInfo}
		return
	}
	for name, on := range selected {
		if name == allMsg || !on {
			continue
		}
		// Read the manifest for the package. A manifest may not exist yet,
		// so create it.
		ic.parseSubpackage(name, projInfo)
	}
}

func (ic *installContext) parseSubpackage(name string, projInfo *module.PackageInfo) {
}

func (ic *installContext) parseInput(inp string) (pkg Package) {
	if len(inp) == 0 {
		cli.Failure("The provided package can't be an empty string")
	}
	// 1. Get the source from the syntax used (see [LongDescription])
	// 	- URL or anything else: Git package
	//  - 'npm:*' - Install from NPM
	//  - './*' - Workspace or local, depending on path
	nameAndVersion := inp
	switch {
	case strings.HasPrefix(inp, "npm:"):
		nameAndVersion = inp[4:]
		pkg = &npmPackage{nameAndVersion: nameAndVersion}
		if nameAndVersion == "" {
			cli.Failure("The name of an NPM dependency can't be empty")
		}
		ic.Debug("Detected NPM package", slog.String("name", nameAndVersion))
	case inp[0] == '.', inp[0] == '/', inp[0] == '\\':
		pkg = &localPackage{path: inp}
		ic.Debug("Detected local package", slog.String("name", nameAndVersion))
	default:
		// It may just be a name if there is no dot (for a URL). It will be
		// treated as a Git package, but we'll warn the user later
		// (after the version is split -- the version may contain a dot)
		pkg = &gitPackage{url: inp}
		ic.Debug("Detected Git package", slog.String("name", nameAndVersion))
	}
	// 2. Version spec: Not allowed for local/workspace packages
	// For NPM packages, '@' may be a scope. The version will be handled by
	// the external package manager.
	//
	// TODO: In the future, we could treat local packages as Git repos and
	// versions can be allowed.
	if name, ver, hasVersion := splitVersion(nameAndVersion); hasVersion &&
		pkg.Source() != npmSource {
		if pkg.Source() == localSource || pkg.Source() == workspaceSource {
			cli.Failuref(
				"Can't provide a version for local/workspace dependeny %s",
				name,
			)
		}
		if ver == "" {
			cli.Failuref("A version is required after %q", klarerrs.Quote(name+"@"))
		}
		spec, err := version.ParseSpecifier(ver)
		if err != nil {
			cli.FailureDetailf(
				"Failed to parse version %q for package %s: ", err.Error(),
				ver, name,
			)
		}
		pkg.setNameAndSpec(name, spec)
	}
	// Show a warning if the package name isn't a path
	if gitPkg, ok := pkg.(*gitPackage); ok && !strings.Contains(gitPkg.url, ".") {
		ic.Warn("Non-absolute Git package detected", slog.String("name", gitPkg.url))
		cli.Warnf(
			"The provided package %s doesn't look like a URL.",
			"It will be treated as a package from Git.\n"+
				" - If this is what you want, pass the full URL to the package's repo.\n"+
				" - To install from NPM, use %s instead.\n"+
				" - Or for a local package, ensure the path starts with './' (e.g. %s)",
			klarerrs.Quote(gitPkg.url),
			klarerrs.Quote("npm:"+inp), klarerrs.Quote("./"+inp),
		)
		if strings.HasPrefix(gitPkg.url, "@") {
			cli.Hint("I think it is an NPM package. We'll install from NPM instead.")
			pkg = &npmPackage{nameAndVersion: nameAndVersion}
		}
	}
	return pkg
}

func splitVersion(name string) (name_, version string, hasVer bool) {
	withoutScopeAt, hasScope := strings.CutPrefix(name, "@")
	name, version, hasVer = strings.Cut(withoutScopeAt, "@")
	if hasScope {
		name = "@" + name
	}
	return name, version, hasVer
}
