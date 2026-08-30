package lsp

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/util/manifest"
)

type Package struct {
	*module.PackageInfo
	deps build.Deps
}

type Module struct {
	*analysis.Module
	PkgPath       string
	compilerInput *build.Input
	// All files with diagnostics when this module was compiled. Includes
	// files in this module.
	depsWithDiags map[string]struct{}
}

func (s *Server) loadPackageFor(filePath string) {
	file := s.fs.Files[filePath]
	if file.Klar.ModulePath != "" {
		return // Already loaded
	}
	// Initialize the file's package
	var (
		pkgDir, _    = module.PackageRoot(filePath)
		isSingleFile = pkgDir == filePath
		pkg          = &Package{}
	)
	file.Klar.ModulePath = filePath // For single-file modules
	if !isSingleFile {
		pkg.PackageInfo = manifest.GetPackageInfo(pkgDir)
		file.Klar.ModulePath = filepath.Dir(filePath)
	}
	s.pkgs[pkgDir] = pkg

	// Initialize its module
	if _, ok := s.modules[file.Klar.ModulePath]; !ok {
		// First time the module is opened
		s.modules[file.Klar.ModulePath] = &Module{PkgPath: pkgDir}
	}
	// This is initialized when the module is, but the typechecked module
	// may exist before this (from compiling this as a dependency)
	if mod := s.modules[file.Klar.ModulePath]; mod.compilerInput == nil {
		inp := &build.Input{
			Kind:    build.KindFile,
			Path:    file.Klar.ModulePath,
			PkgInfo: pkg.PackageInfo,
		}
		if !isSingleFile {
			inp.Kind = build.KindModule
			inp.Manifest = pkg.Manifest
			inp.Lockfile = s.readLockfile(pkg.PackageInfo.ProjectDir)
		}
		mod.compilerInput = inp
	}
	file.Klar.Module = s.modules[file.Klar.ModulePath]

	s.Info(
		"Package loaded",
		slog.String("file", filePath), slog.String("packagePath", pkgDir),
	)
	if isSingleFile {
		s.Info("File isn't in a package", slog.String("path", filePath))
	}
}

func (s *Server) readLockfile(projDir string) *glaslock.Lockfile {
	lockfilePath := filepath.Join(projDir, module.LockFile)
	f, err := os.Open(lockfilePath)
	if err != nil {
		return nil // No lockfile
	}
	defer f.Close()
	lockfile, err := glaslock.Parse(f)
	if err != nil {
		// TODO: Handle incompatible version errors by running `glas install`
		s.Error(
			"Failed to parse lockfile",
			slog.String("path", lockfilePath), slog.Any("error", err),
		)
	}
	return lockfile
}
