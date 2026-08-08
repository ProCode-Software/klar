package add

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/cmd/glas/internal/monorepo"
	"github.com/ProCode-Software/klar/cmd/glas/internal/spinner"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/prompt"
	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
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
	Source() glaslock.PackageSource
	ResolvedVersion() string
	KlarVersion() version.Version // Version converted to Klar
	Info(*installContext) *pkgInfo
	Install(*installContext)
}

const (
	GitSource       = glaslock.Git
	NPMSource       = glaslock.NPM
	LocalSource     = glaslock.Local
	WorkspaceSource = glaslock.Workspace
)

type installContext struct {
	targetPkgs []*module.PackageInfo
	*slog.Logger
	*argparse.Parser
	isInteractive bool
	debug         bool
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
	startTime := time.Now()

	if len(pkgs) > 1 && p.Flag("subdir").Set {
		cli.Failure("The '--subdir' flag can be used with only 1 dependency")
	}

	// Setup verbose logging
	logger, err := util.SetLogger(p.Flag("verbose").Bool(), false)
	if err != nil {
		cli.FailureError(err)
	}
	ic.Logger = logger
	if ic.Logger.Enabled(context.Background(), slog.LevelDebug) {
		ic.debug = true
	}
	ic.isInteractive = term.IsTerminal(int(os.Stdout.Fd()))

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
		if p.Flag("yes").Bool() || p.Flag("project").Bool() || !ic.isInteractive {
			// Assume they're installing for the whole project
			ic.targetPkgs = []*module.PackageInfo{projInfo}
			break
		}
		ic.promptSpecificPackages(projInfo)
	default:
		ic.targetPkgs = []*module.PackageInfo{projInfo}
	}

	parsedPkgs := make([]pkgsAndInfo, len(pkgs))
	// 2. Now that we have the target packages loaded, we can finally install
	// each package. Do this in parallel because of NPM requests, and cloning
	// Git repos (shallow until confirmed).
	var wg sync.WaitGroup
	for i, inp := range pkgs {
		wg.Go(func() {
			defer cli.HandleSignalExit() // When [cli.Failure] is called
			pkg := ic.parseInput(inp)
			parsedPkgs[i] = pkgsAndInfo{pkg, pkg.Info(ic)}
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

	// 3. Ask for confirmation, displaying the info for each package
	rejected := ic.promptPackages(parsedPkgs)

	// 4. Install accepted packages
	for i, pair := range parsedPkgs {
		if _, ok := rejected[i]; ok {
			continue
		}
		pair.pkg.Install(ic)
	}

	// 5. Update lockfile and manifest
	lockfile, lockPath := ic.getLockfile()
	for _, pair := range parsedPkgs {
		ic.updateManifests(pair.pkg)
		ic.updateLockfile(pair.pkg, lockfile)
	}
	ic.writeLockfile(lockfile, lockPath)
	ic.writeManifests()

	// 6. Summary
	elapsed := time.Since(startTime)
	ansi.TagPrintfln(
		"\n<** g!>🚚 Successfully installed <c>%s</c></> in <c>%s</c>",
		klarerrs.FormatCount(len(parsedPkgs), "package"),
		util.FormatDuration(elapsed),
	)
	for _, pair := range parsedPkgs {
		ansi.TagPrintfln(" <g>+</g> %s <d>%s</d>", pair.info.name, pair.info.version)
	}
	// TODO: Show documentation links
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
		pkg.Source() != NPMSource {
		if pkg.Source() == LocalSource || pkg.Source() == WorkspaceSource {
			cli.Failuref(
				"Can't provide a version for local/workspace dependency %s",
				name,
			)
		}
		if ver == "" {
			cli.Failuref("A version is required after %s", klarerrs.Quote(name+"@"))
		}
		// For Git packages, the specifier may be:
		// - A commit '+...'
		// - A tag/version specifier 'v...'
		// - A branch '...'
		gitPkg := pkg.(*gitPackage)
		gitPkg.url, gitPkg.rawSpec = name, ver
		switch ver[0] {
		case 'v':
			gitPkg.specKind = glaspack.TagRef
		case '+':
			gitPkg.specKind = glaspack.CommitRef
			gitPkg.rawSpec = ver[1:]
		default:
			gitPkg.specKind = glaspack.BranchRef
		}
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

type pkgsAndInfo struct {
	pkg  Package
	info *pkgInfo
}

func (ic *installContext) promptPackages(parsedPkgs []pkgsAndInfo) (rejected map[int]struct{}) {
	rejected = make(map[int]struct{})
	// 3. Ask for confirmation
	for i, pair := range parsedPkgs {
		yes := ic.Flag("yes").Bool()
		if pair.info == nil {
			continue
		}
		if len(parsedPkgs) > 1 {
			ansi.ColorPrintfln(
				ansi.CodeDim+ansi.CodeUnderline, "Package %d of %d", i+1, len(parsedPkgs),
			)
		}
		showInfo(pair.pkg, pair.info)
		// Also auto-confirm if not interactive / pipe
		if !yes && ic.isInteractive {
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
	return rejected
}

func splitVersion(name string) (name_, version string, hasVer bool) {
	withoutScopeAt, hasScope := strings.CutPrefix(name, "@")
	name, version, hasVer = strings.Cut(withoutScopeAt, "@")
	if hasScope {
		name = "@" + name
	}
	return name, version, hasVer
}
