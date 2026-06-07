package build

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/pkg/klon"
)

const sep = string(filepath.Separator)

type ProjectInput struct {
	Path       string
	Kind       InputKind
	Manifest   *glaspack.Manifest
	ProjectDir string
	PkgInfo    *module.PackageInfo
	Config     *klarbuild.File
}

// IsKlarFile returns true if file's extension is '.klar' or it doesn't have an extension.
func IsKlarFile(file string) bool {
	return strings.HasSuffix(file, ".klar") || filepath.Ext(file) == ""
}

func IsTestFile(file string) bool {
	return strings.HasSuffix(file, ".test.klar")
}

// klarBuildMode == 0: ResolveInput looks for a klar.build
// klarBuildMode == 1: No klar.build is resolved. The caller is expected
// to parse the klar.build and set i.Config.
// klarBuildMode == 2: ResolveInput uses the default klar.build
func (pc *ProjectCompiler) ResolveInput(s string,
	klarBuildMode int, isFromKlarBuild bool,
) (i *ProjectInput, err error) {
	// If the input refers to a module import path (`@...`), use the manifest for the cwd
	switch {
	case s == "-":
		// Input is a file from stdin
		// Don't use a manifest
		return &ProjectInput{Kind: KindStdin}, nil
	case s[0] == '@':
		// If the input refers to a module import path (`@...`), use the manifest for the cwd
		panic("module import paths are not supported yet")
	default:
	}
	s = pc.Abs(s)
	if info, err := os.Stat(s); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			kind := "input"
			if isFromKlarBuild {
				kind += " from klar.build"
			}
			cli.ErrNotFound(s, kind)
		}
		return nil, &FilesystemError{"stat", s, err}
	} else if info.IsDir() {
		// Directory: module or package
		i = &ProjectInput{Path: s, Kind: KindModule}
		if module.IsPackage(s) {
			i.Kind = KindPackage
		}
	} else {
		// File
		if !IsKlarFile(s) {
			return nil, &InterfaceError{Code: ErrNotAKlarFile, Value: s}
		}
		i = &ProjectInput{Path: s, Kind: KindFile}
	}
	// Resolve the manifest and get the PackageInfo
	if i.Kind != KindStdin && i.Manifest == nil {
		if err := i.ResolveManifest(pc.Compiler); err != nil {
			return nil, err
		}
	}
	// Find the klar.build file
	switch klarBuildMode {
	case 0:
		if configPath := i.ResolveKlarBuild(); configPath != "" {
			var warn []*klon.Error
			if i.Config, warn, err = klarbuild.Parse(configPath); err != nil {
				return nil, err
			}
			pc.PrintKlonWarnings(warn, configPath)
			break
		}
		fallthrough
	case 2:
		i.Config = klarbuild.Default()
	case 1:
		// A forced klar.build will be parsed by the caller
	}
	return i, nil
}

func (i *ProjectInput) ResolveManifest(c *Compiler) error {
	dir := i.Path
	if i.Kind == KindFile {
		dir = filepath.Dir(dir)
	}
	man, projDir, err := i.resolveManifest(dir, c)
	if err != nil {
		return err
	}
	i.Manifest = man
	i.PkgInfo = module.NewPackageInfo(projDir, man)
	i.ProjectDir = projDir
	return nil
}

func (i *ProjectInput) ResolveKlarBuild() (path string) {
	dir := i.Path
	if i.Kind == KindFile {
		dir = filepath.Dir(i.Path)
	}
	checkDir := func(dir string) bool {
		klarBuild := dir + sep + module.BuildFile
		if _, err := os.Stat(klarBuild); err == nil {
			path = klarBuild
			return true
		}
		return false
	}
	// Look inside directory before outside
	if checkDir(dir) {
		return
	}
	_, projRoot := module.PackageRoot(dir)
	if dir == projRoot || checkDir(projRoot) { // Check project root
		return
	} else if i.Kind == KindFile {
		return // For projectless input: don't look outside
	}
	for {
		// Stop after project directory (i.Path may be a project directory)
		if _, ok := module.KlarProjectDirs[filepath.Base(dir)]; ok {
			return
		}
		parent := filepath.Dir(dir)
		if dir == parent {
			return // At root
		}
		dir = parent
		if checkDir(dir) || dir == projRoot /* Stop at project root */ {
			return
		}
	}
}

var manifestCache map[string]*glaspack.Manifest

// resolveManifest resolves the manifest located in dir and parses it.
func (i *ProjectInput) resolveManifest(dir string, c *Compiler) (
	m *glaspack.Manifest, projDir string, err error,
) {
	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}
	newKlonError := func(err error, path string) *InterfaceError {
		return &InterfaceError{Code: ErrInvalidConfig, Err: err, Value: path}
	}
	pkgDir, projDir := module.PackageRoot(dir)
	var (
		pkgFile  = filepath.Join(pkgDir, module.ManifestFile)
		projFile = filepath.Join(projDir, module.ManifestFile)
		warn     []*klon.Error
		ok       bool
	)
	if m, ok = manifestCache[projFile]; !ok && exists(pkgFile) {
		m, warn, err = glaspack.Parse(pkgFile)
		if err != nil {
			return nil, projDir, newKlonError(err, pkgFile)
		}
		manifestCache[projFile] = m
		c.PrintKlonWarnings(warn, pkgFile)
	}
	if pkgDir == projDir || !exists(projFile) {
		// Make sure at least one manifest exists
		if m == nil {
			cli.ErrNoManifest(pkgDir)
		}
		return m, projDir, nil
	}
	// Check cache for project manifest
	if m2, ok := manifestCache[projFile]; ok {
		m, err = glaspack.Merge(m, m2)
		return m, projDir, err
	}
	// Project-level manifest
	m2, warn, err := glaspack.Parse(projFile)
	if err != nil {
		return nil, projDir, newKlonError(err, projFile)
	}
	manifestCache[projFile] = m2
	c.PrintKlonWarnings(warn, projFile)
	m, err = glaspack.Merge(m, m2)
	return m, projDir, err
}
