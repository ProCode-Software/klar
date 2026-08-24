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
}

func (s *Server) loadPackageFor(fileURI string) {
	file := s.fs.Files[fileURI]
	if file.ModulePath != "" {
		return // Already loaded
	}
	// Initialize the file's package
	var (
		pkgDir, _    = module.PackageRoot(fileURI)
		isSingleFile = pkgDir == fileURI
		pkg          = &Package{}
	)
	file.ModulePath = fileURI // For single-file modules
	if !isSingleFile {
		pkg.PackageInfo = manifest.GetPackageInfo(pkgDir)
		file.ModulePath = filepath.Dir(fileURI)
	}
	s.pkgs[pkgDir] = pkg

	// Initialize its module
	if _, ok := s.modules[file.ModulePath]; !ok {
		// First file in the module loaded
		inp := &build.Input{
			Kind:    build.KindModule,
			Path:    file.ModulePath,
			PkgInfo: pkg.PackageInfo,
		}
		if pkg.PackageInfo != nil {
			inp.Manifest = pkg.Manifest
			inp.Lockfile = s.readLockfile(pkg.PackageInfo.ProjectDir)
		}
		s.modules[file.ModulePath] = &Module{PkgPath: pkgDir, compilerInput: inp}
	}
	file.Module = s.modules[file.ModulePath]

	s.Info(
		"Package loaded",
		slog.String("file", fileURI), slog.String("packagePath", pkgDir),
	)
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
