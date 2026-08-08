package add

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/version"
)

func (ic *installContext) updateManifests(pkg Package) {
	for _, target := range ic.targetPkgs {
		man := target.Manifest
		entry := glaspack.DependencyCoder{}
		switch pkg := pkg.(type) {
		case *gitPackage:
			var verSpec version.Specifier
			if pkg.rawSpec != "" {
				var err error
				if verSpec, err = version.ParseSpecifier(pkg.rawSpec); err != nil {
					// Error should have been caught during installation
					panic(fmt.Sprintf("version.ParseSpecifier failed: %v", verSpec))
				}
			} else {
				verSpec = version.MinimumSpecifier(pkg.manifest.Version)
			}
			entry.DependencySpecifier = &glaspack.GitSpecifier{
				URL:        pkg.url,
				Subpackage: pkg.subPath(),
				RefKind:    pkg.specKind,
				Ref:        pkg.shortCommit(),
				Version:    &verSpec,
			}
		case *npmPackage:
			entry.DependencySpecifier = &glaspack.NPMSpecifier{
				Name: pkg.Name(),
				// TODO: Specifier converted from SemVer
			}
		case *localPackage:
			if pkg.isWorkspace {
				entry.DependencySpecifier = &glaspack.WorkspaceSpecifier{
					Subpackage: pkg.workspacePkgName(),
				}
				break
			}
			entry.DependencySpecifier = &glaspack.LocalSpecifier{
				Path: pkg.path,
			}
		}
		if ic.Flag("dev").Bool() {
			man.DevelopmentDependencies = append(man.DevelopmentDependencies, entry)
		} else {
			man.Dependencies = append(man.Dependencies, entry)
		}
	}
}

func (ic *installContext) updateLockfile(pkg Package, lockfile *glaslock.Lockfile) {
	lockPkg := &glaslock.Package{
		PackageHeader: glaslock.PackageHeader{
			Name:    pkg.Name(),
			Version: pkg.KlarVersion(),
			From:    pkg.Source(),
		},
	}
	// TODO: Add 'for workspaces' and dependencies to lockfile package
	if ic.Flag("dev").Bool() {
		lockPkg.DevOnly = true
	}
	addLockfileInfo(pkg, lockPkg)
	lockPkg.PackageHeader.GenerateHash() // Hash after adding Git commit to header
	if _, ok := lockfile.PackageMap[lockPkg.Hash]; !ok {
		lockfile.Packages = append(lockfile.Packages, lockPkg)
		lockfile.PackageMap[lockPkg.Hash] = lockPkg
	} else {
		// Use [slog.Any] to avoid unnecessary Stringing if logging is disabled
		ic.Info("Header already in lockfile", "header", lockPkg.PackageHeader)
	}
}

func addLockfileInfo(pkg Package, lockPkg *glaslock.Package) {
	switch pkg := pkg.(type) {
	case *gitPackage:
		lockPkg.PackageHeader.GitCommit = pkg.shortCommit()
		lockPkg.Info = &glaslock.GitInfo{
			RefType:   pkg.specKind,
			Ref:       pkg.rawSpec,
			URL:       pkg.url,
			Subpath:   pkg.subPath(),
			Integrity: pkg.rev,
		}
	case *npmPackage:
		// TODO: Integrity
		lockPkg.Info = &glaslock.NPMInfo{Registry: npmRegistry, Integrity: ""}
	case *localPackage:
		if pkg.isWorkspace {
			lockPkg.Info = &glaslock.WorkspaceInfo{Dir: pkg.workspacePkgName()}
			break
		}
		lockPkg.Info = &glaslock.LocalInfo{Path: pkg.path}
	}
}

func (ic *installContext) getLockfile() (lockfile *glaslock.Lockfile, lockPath string) {
	// TODO: One advantage of the line-based lockfile format is that it's feasible to
	// append to without parsing. The only disadvantage is that appending may create
	// duplicate packages. In the future, consider appending to the lockfile.
	//
	// Possible ways to detect duplicates:
	//  A. strings.Contains, but a comment may contain the hash
	//  B. Keep a key at the bottom of the lockfile with a list of all hashes
	lockPath = filepath.Join(ic.targetPkgs[0].ProjectDir, module.LockFile)
	pathAttr := slog.String("path", lockPath)
	if _, err := os.Stat(lockPath); err == nil {
		// If there's an error, we're just going to create a new lockfile.
		// 'glas install' will need to be run later.
		if existingLockfile, err := glaslock.ParseFile(lockPath); err == nil {
			lockfile = existingLockfile
			ic.Logger.Debug("Existing lockfile parsed", pathAttr)
		} else {
			ic.Logger.Warn("Lockfile has errors, recreating", pathAttr)
		}
	}
	if lockfile == nil {
		lockfile = glaslock.NewLockfile(cli.ParsedKlarVersion)
		ic.Logger.Info("Creating new lockfile", pathAttr)
	}
	return lockfile, lockPath
}

func (ic *installContext) writeLockfile(lockfile *glaslock.Lockfile, lockPath string) {
	f, err := os.Create(lockPath)
	if err != nil {
		cli.FailureDetailf("Failed to create lockfile at %s: ", err.Error(), lockPath)
	}
	checkWriteError := func(err error) {
		if err != nil {
			cli.FailureDetailf("Failed to write lockfile at %s: ", err.Error(), lockPath)
		}
	}
	// Can't 'checkWriteError(f.Close())' because it closes immediately
	defer func() { checkWriteError(f.Close()) }()

	_, err = lockfile.WriteWithDisclaimerTo(f)
	checkWriteError(err)
	ic.Logger.Debug("Lockfile written", slog.String("path", lockPath))
}

func (ic *installContext) writeManifests() {
	for _, target := range ic.targetPkgs {
		man := target.Manifest
		manPath := filepath.Join(target.Dir, module.ManifestFile)
		// TODO: Write manifest. It may have to be created for the specific package.
		// After Klon marshalling is implemented
		_ = man
		ic.Debug("Updated manifest", slog.String("path", manPath))
	}
}
