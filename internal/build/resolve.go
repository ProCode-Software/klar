package build

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/pkg/klon"
)

const sep = string(filepath.Separator)

type Input struct {
	Path      string
	Kind      InputKind
	Manifest  *glaspack.Manifest
	PkgInfo   *module.PackageInfo
	Lockfile  *glaslock.Lockfile
	KlarBuild *klarbuild.File
	Targets   []target.Target
}

func (i *Input) IsSingleFile() bool {
	return i.Kind == KindFile || i.Kind == KindStdin
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
func (c *Compiler) ResolveInput(s string,
	klarBuildMode int, isFromKlarBuild bool,
) (i *Input, err error) {
	// If the input refers to a module import path (`@...`), use the manifest for the cwd
	switch {
	case s == "-":
		// Input is a file from stdin
		// Don't use a manifest
		return &Input{Kind: KindStdin}, nil
	case s[0] == '@':
		// If the input refers to a module import path (`@...`), use the manifest for the cwd
		panic("module import paths are not supported yet")
	default:
	}
	s = c.Abs(s)
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
		// Ensure the user isn't passing the 'test' folder as an input outside of test mode
		if c.Mode != ModeTest && info.Name() == module.TestDir {
			return nil, &InterfaceError{Code: ErrTestInput, Value: s}
		}
		// Directory: module or package
		i = &Input{Path: s, Kind: KindModule}
		if module.IsPackage(s) {
			i.Kind = KindPackage
		}
	} else {
		// File
		if !IsKlarFile(s) {
			return nil, &InterfaceError{Code: ErrNotAKlarFile, Value: s}
		}
		i = &Input{Path: s, Kind: KindFile}
	}
	// Resolve the manifest and get the PackageInfo
	if i.Kind != KindStdin && i.Manifest == nil {
		if err := i.ResolveManifest(c); err != nil {
			return nil, err
		}
	}
	// Find the klar.build file
	switch klarBuildMode {
	case 0:
		if configPath := i.ResolveKlarBuild(); configPath != "" {
			var warn []*klon.Error
			if i.KlarBuild, warn, err = klarbuild.Parse(configPath); err != nil {
				return nil, err
			}
			c.PrintKlonWarnings(warn, configPath)
			break
		}
		fallthrough
	case 2:
		i.KlarBuild = klarbuild.Default()
	case 1:
		// A forced klar.build will be parsed by the caller
	}
	// Set the input's target
	// klar.build's Target has priority over the manifest's
	if i.KlarBuild != nil {
		i.Targets = []target.Target{i.KlarBuild.Target}
	} else if i.Manifest != nil {
		i.Targets = i.Manifest.Target
	}
	return i, nil
}

func (i *Input) ResolveManifest(c *Compiler) error {
	dir := i.Path
	if i.Kind == KindFile {
		dir = filepath.Dir(dir)
	}
	man, pkgDir, projDir, err := i.resolveManifest(dir, c)
	if err != nil {
		return err
	}
	i.Manifest = man
	i.PkgInfo = module.NewPackageInfo(pkgDir, projDir, man)
	return nil
}

func (i *Input) ResolveKlarBuild() (path string) {
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
func (i *Input) resolveManifest(dir string, c *Compiler) (
	m *glaspack.Manifest, pkgDir, projDir string, err error,
) {
	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}
	newKlonError := func(err error, path string) *InterfaceError {
		return &InterfaceError{Code: ErrInvalidConfig, Err: err, Value: path}
	}
	pkgDir, projDir = module.PackageRoot(dir)
	var (
		pkgFile  = filepath.Join(pkgDir, module.ManifestFile)
		projFile = filepath.Join(projDir, module.ManifestFile)
		warn     []*klon.Error
		ok       bool
	)
	if manifestCache == nil {
		manifestCache = make(map[string]*glaspack.Manifest)
	}
	if m, ok = manifestCache[projFile]; !ok && exists(pkgFile) {
		m, warn, err = glaspack.Parse(pkgFile)
		if err != nil {
			return nil, pkgDir, projDir, newKlonError(err, pkgFile)
		}
		manifestCache[projFile] = m
		c.PrintKlonWarnings(warn, pkgFile)
	}
	if pkgDir == projDir || !exists(projFile) {
		// Make sure at least one manifest exists
		if m == nil {
			cli.ErrNoManifest(pkgDir)
		}
		return m, pkgDir, projDir, nil
	}
	// Check cache for project manifest
	if m2, ok := manifestCache[projFile]; ok {
		m, err = glaspack.Merge(m, m2)
		return m, pkgDir, projDir, err
	}
	// Project-level manifest
	m2, warn, err := glaspack.Parse(projFile)
	if err != nil {
		return nil, pkgDir, projDir, newKlonError(err, projFile)
	}
	manifestCache[projFile] = m2
	c.PrintKlonWarnings(warn, projFile)
	m, err = glaspack.Merge(m, m2)
	return m, pkgDir, projDir, err
}

func (ld *Loader) ResolveInputModules() (modules []*Module, klarFiles int, err error) {
	switch ld.Kind {
	case KindPackage:
		ld.Info("Resolving package", slog.String("path", ld.Path))
		modules, klarFiles, err = ld.resolvePackage(ld.Path, false)
	case KindFile:
		name := filepath.Base(ld.Path)
		if ld.Mode != ModeTest && IsTestFile(name) {
			return nil, 0, &InterfaceError{Code: ErrTestInput, Value: ld.Path}
		}
		ld.Info("Resolved file", slog.String("path", ld.Path))
		return []*Module{{
			Path:       ld.Path,
			Programs:   map[string]*ast.Program{name: nil},
			SingleFile: true,
		}}, 1, nil
	case KindStdin:
		ld.Info("Resolved file from stdin")
		// Empty paths are stdin
		return []*Module{{
			Path:       "",
			Programs:   map[string]*ast.Program{"": nil},
			SingleFile: true,
		}}, 1, nil
	case KindModule:
		ld.Info("Resolving module", slog.String("modulePath", ld.Path))
		klarFiles, err = ld.moduleFromDir(ld.Path, &modules, 0)
	}
	if klarFiles == 0 && err == nil {
		err = &InterfaceError{Code: ErrNoKlarFiles, Value: ld.Path}
	}
	return
}

// moduleFromDir reads the contents of dir, assigning dir to a [*File] with
// dir's contents groupd by submodules (directories), Klar files, and assets
// (non-Klar files). moduleFromDir reports an error if it encounters a
// submodule when depth is [MaxModuleDepth], per the Klar Project Structure spec.
func (c *Compiler) moduleFromDir(
	dir string, modules *[]*Module, depth int,
) (klarFiles int, err error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return klarFiles, &FilesystemError{"read", dir, err}
	}
	m := &Module{
		Path:     dir,
		Programs: make(map[string]*ast.Program, len(items)),
	}
	*modules = append(*modules, m)
	for _, d := range items {
		name := d.Name()
		path := dir + sep + name
		switch {
		case d.IsDir(): // Submodule
			if depth >= module.MaxModuleDepth {
				err = &InterfaceError{Value: path, Code: ErrMaxModuleDepth}
				return
			}
			c.Info("Resolving submodule", slog.String("modulePath", path))
			n, err := c.moduleFromDir(path, modules, depth+1)
			klarFiles += n
			if err != nil {
				return klarFiles, err
			}
		case IsTestFile(name):
			if filepath.Base(dir) != module.TestDir {
				// Test files must be in test/ folders
				// TODO: Should this remain a requirement?
				err = &InterfaceError{Value: path, Code: ErrMisplacedTest}
				return
			}
			fallthrough
		case IsKlarFile(name):
			m.Programs[name] = nil
			klarFiles++
		default:
			// TODO: filter by klar.build's asset extensions
			m.Assets = append(m.Assets, name)
		}
	}
	return
}

func (c *Compiler) resolvePackage(path string, nesting bool) (
	modules []*Module, klarFiles int, err error,
) {
	defer func() {
		if err != nil {
			c.Error(
				"Error while resolving package",
				slog.String("package", path), slog.Any("error", err),
			)
		}
	}()
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, klarFiles, &FilesystemError{"read", path, err}
	}
	for _, d := range items {
		name := d.Name()
		fullPath := path + sep + name
		if !d.IsDir() {
			// Report an error if there's a Klar file directly in a package
			// Extensionless files are still allowed
			if strings.HasSuffix(name, ".klar") {
				err = &InterfaceError{Value: fullPath, Code: ErrFileInRoot}
				return
			}
			continue
		}
		switch name {
		case module.PkgDir: // pkg
			/* if nesting {
				err = &InterfaceError{Value: fullPath, Code: ErrNestedKlarFolder}
				return
			}
			pkgs, err := os.ReadDir(fullPath)
			if err != nil {
				return modules, klarFiles, &FilesystemError{"read", fullPath, err}
			}
			for _, pkg := range pkgs {
				fullPkg := fullPath + sep + pkg.Name()
				if !pkg.IsDir() {
					return modules, klarFiles, &InterfaceError{Code: ErrFileInPkgDir, Value: fullPkg}
				}
				n, err := c.resolvePackage(fullPkg, modules, true, info)
				klarFiles += n
				if err != nil {
					c.Error(
						"Failed to resolve subpackage",
						slog.String("package", fullPkg), slog.Any("error", err),
					)
					return klarFiles, err
				}
			} */
			// TODO: Move
		case module.LocalDataDir: // .klar
			if nesting {
				err = &InterfaceError{Value: fullPath, Code: ErrNestedKlarFolder}
				return
			}
		case module.SharedDir: // shared
			if nesting {
				err = &InterfaceError{Value: fullPath, Code: ErrNestedKlarFolder}
				return
			}
			fallthrough
		case module.CmdDir, module.TestDir: // cmd, test
			if name == module.TestDir && c.Mode != ModeTest {
				break // Load test folder only in test mode
			}
			// The only Klar project directories that contain buildable modules,
			// inclding shared/, other than src/
			c.Info("Resolving modules in", slog.String("path", fullPath))
			n, err := c.moduleFromDir(fullPath, &modules, 0)
			klarFiles += n
			if err != nil {
				return modules, klarFiles, err
			}
		case module.SrcDir: // src
			n, err := c.resolveSrcDir(fullPath, &modules)
			klarFiles += n
			if err != nil {
				return modules, klarFiles, err
			}
		default:
			// Other directory: ignore
			c.Info("Ignoring top-level directory", slog.String("dir", fullPath))
			continue
		}
	}
	return modules, klarFiles, nil
}

func (c *Compiler) resolveSrcDir(dir string, modules *[]*Module) (klarFiles int, err error) {
	var srcMod *Module // Initialized only if there are assets in the src folder

	items, err := os.ReadDir(dir)
	if err != nil {
		return klarFiles, &FilesystemError{"read", dir, err}
	}
	for _, d := range items {
		name := d.Name()
		path := dir + sep + name
		switch {
		case d.IsDir(): // Module
			c.Info("Resolving module", slog.String("modulePath", path))
			n, err := c.moduleFromDir(path, modules, 0)
			klarFiles += n
			if err != nil {
				return klarFiles, err
			}
		case IsTestFile(name):
			err = &InterfaceError{Value: path, Code: ErrMisplacedTest}
			return
		case IsKlarFile(name):
			err = &InterfaceError{Value: path, Code: ErrFileInRoot}
			return
		default:
			// Asset in the src folder
			// TODO: We will allow it, but we currently don't know how it
			// will be handled
			if srcMod == nil {
				srcMod = &Module{Path: dir}
				*modules = append(*modules, srcMod)
			}
			srcMod.Assets = append(srcMod.Assets, name)
		}
	}
	return
}
