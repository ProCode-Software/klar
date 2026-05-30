package build

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/module"
)

const sep = string(filepath.Separator)

// Step 1: Determine the kind of each input and resolve its klar.build file
// =====

// ResolveInputs finds the location, kind, and klar.build file for each input.
// ResolveInputs returns an error if a path cannot be read, an input is invalid,
// or an input is not a .klar file.
func ResolveInputs(inputs []string, klarBuildPath string) ([]Input, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	res := make([]Input, 0, len(inputs))
	klarBuildDir := filepath.Dir(klarBuildPath)
	for _, input := range inputs {
		if len(input) == 0 {
			continue
		}
		var i Input
		switch {
		case input == "-": // stdin
			i = Input{Kind: KindStdin}
		case input[0] == '@':
			// Module name reference: @foo.bar
			if len(input) == 1 {
				return nil, &InterfaceError{Code: ErrModuleDescriptor}
			}
			// TODO: resolve by import path
			i = Input{Kind: KindModule, Name: input}
		default:
			if !filepath.IsAbs(input) {
				input = filepath.Join(klarBuildDir, input)
			}
			info, err := os.Stat(input)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					kind := "input"
					if klarBuildPath != "" {
						kind += " from klar.build"
					}
					cli.ErrNotFound(input, kind)
				}
				return nil, &FilesystemError{"stat", input, err}
			}
			fullPath, err := filepath.Abs(input)
			if err != nil {
				return nil, &FilesystemError{"expand path of", input, err}
			}
			if !info.IsDir() { // File
				if !IsKlarFile(input) {
					return nil, &InterfaceError{Code: ErrNotAKlarFile, Value: input}
				}
				i = Input{Path: fullPath, Kind: KindFile}
			} else {
				// Directory: module or package
				i = Input{Path: fullPath, Name: filepath.Base(fullPath), Kind: KindModule}
				if module.IsPackage(fullPath) {
					i.Kind = KindPackage
				}
			}
			// Get path to closest klar.build file
			if i.KlarBuild = klarBuildPath; klarBuildPath == "" {
				ResolveKlarBuild(&i)
			}
		}
		res = append(res, i)
	}
	return res, nil
}

// IsKlarFile returns true if file's extension is '.klar' or it doesn't have an extension.
func IsKlarFile(file string) bool {
	return strings.HasSuffix(file, ".klar") || filepath.Ext(file) == ""
}

// ResolveKlarBuild sets i's [Input.KlarBuild] to the closest 'klar.build' file
// to its [Input.Path], if it is found. Otherwise, ResolveKlarBuild does nothing.
func ResolveKlarBuild(i *Input) {
	dir := i.Path
	if i.Kind == KindFile {
		dir = filepath.Dir(i.Path)
	}
	checkDir := func(dir string) bool {
		klarBuild := dir + sep + module.BuildFile
		if _, err := os.Stat(klarBuild); err == nil {
			i.KlarBuild = klarBuild
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

// Step 2: Create the map of inputs to modules. During this step, all packages,
// modules, assets, and files are resolved.
// =====

// Per Project Structure Spec: No more than 4 parts of a module
const MaxModuleDepth = 4

// ResolveModules groups all inputs from all [Compiler.Options] into modules.
// An error is returned when ResolveModules encounters an error while walking
// a module's directories, if [MaxModuleDepth] is exceeded, or if no Klar files
// were found for an input.
func (c *Compiler) ResolveModules() (totalFiles int, err error) {
	c.inputs = make(map[*Input]*InputOptions, len(c.Options))
	c.moduleInputs = make(map[*Module]*InputOptions, len(c.Options))
	c.Modules = make([]*Module, 0, len(c.Options))
	// Show an error if no Klar files to compile were found
	checkFileCount := func(klarFiles int, path string, err *error) {
		if klarFiles == 0 && *err == nil {
			*err = &InterfaceError{Code: ErrNoKlarFiles, Value: path}
		}
		totalFiles += klarFiles
	}
	for _, opt := range c.Options {
		for i := range opt.Inputs {
			inp := &opt.Inputs[i]
			info := &InputOptions{Options: opt}
			// TODO: resolve glas.pack for module/package
			switch inp.Kind {
			case KindPackage:
				c.Info("Resolving package", slog.String("path", inp.Path))
				klarFiles, err := c.resolvePackage(inp.Path, &info.Modules, false, info)
				checkFileCount(klarFiles, inp.Path, &err)
				if err != nil {
					return totalFiles, err
				}
			case KindFile:
				info.Modules = []*Module{{
					Name:       inp.Name,
					Path:       inp.Path,
					SingleFile: true,
					Files:      []string{inp.Path},
				}}
				c.Modules = append(c.Modules, info.Modules[0])
				c.moduleInputs[info.Modules[0]] = info
				totalFiles++
				c.Info("Resolved file", slog.String("path", inp.Path))
			case KindStdin:
				// Empty paths are stdin
				info.Modules = []*Module{{Files: []string{""}, SingleFile: true}}
				c.Modules = append(c.Modules, info.Modules[0])
				c.moduleInputs[info.Modules[0]] = info
				totalFiles++
				c.Info("Resolved file from stdin")
			case KindModule:
				c.Info("Resolving module", slog.String("modulePath", inp.Path))
				klarFiles, err := c.moduleFromDir(
					inp.Name, inp.Path, &info.Modules, 0, info,
				)
				checkFileCount(klarFiles, inp.Path, &err)
				if err != nil {
					return totalFiles, err
				}
			}
			c.inputs[inp] = info
		}
	}
	if totalFiles == 0 {
		return totalFiles, &InterfaceError{Code: ErrNothingToCompile}
	}
	return totalFiles, nil
}

// moduleFromDir reads the contents of dir, assigning dir to a [*File] with
// dir's contents groupd by submodules (directories), Klar files, and assets
// (non-Klar files). moduleFromDir reports an error if it encounters a
// submodule when depth is [MaxModuleDepth], per the Klar Project Structure spec.
func (c *Compiler) moduleFromDir(
	moduleName, dir string, modules *[]*Module, depth int, info *InputOptions,
) (klarFiles int, err error) {
	m := &Module{Name: moduleName, Path: dir}
	c.Modules = append(c.Modules, m)
	*modules = append(*modules, m)
	c.moduleInputs[m] = info

	items, err := os.ReadDir(dir)
	if err != nil {
		return klarFiles, &FilesystemError{"read", dir, err}
	}
	m.Files = make([]string, 0, len(items))
	for _, d := range items {
		name := d.Name()
		path := dir + sep + name
		switch {
		case d.IsDir(): // Submodule
			if depth >= MaxModuleDepth {
				err = &InterfaceError{Value: path, Code: ErrMaxModuleDepth}
				return
			}
			m.Submodules = append(m.Submodules, path)
			c.Info("Resolving submodule", slog.String("modulePath", path))
			n, err := c.moduleFromDir(name, path, modules, depth+1, info)
			klarFiles += n
			if err != nil {
				return klarFiles, err
			}
		case strings.HasSuffix(name, ".test.klar"):
			if moduleName != module.TestDir {
				// Test files must be in test/ folders
				err = &InterfaceError{Value: path, Code: ErrMisplacedTest}
				return
			}
			fallthrough
		case IsKlarFile(name):
			m.Files = append(m.Files, path)
			klarFiles++
		default:
			m.Assets = append(m.Assets, path)
		}
	}
	return
}

func (c *Compiler) resolvePackage(
	path string, modules *[]*Module, nesting bool, info *InputOptions,
) (klarFiles int, err error) {
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
		return klarFiles, &FilesystemError{"read", path, err}
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
			if nesting {
				err = &InterfaceError{Value: fullPath, Code: ErrNestedKlarFolder}
				return
			}
			pkgs, err := os.ReadDir(fullPath)
			if err != nil {
				return klarFiles, &FilesystemError{"read", fullPath, err}
			}
			for _, pkg := range pkgs {
				fullPkg := fullPath + sep + pkg.Name()
				if !pkg.IsDir() {
					return klarFiles,
						&InterfaceError{Code: ErrFileInPkgDir, Value: fullPkg}
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
			}
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
		case module.SrcDir, module.CmdDir, module.TestDir: // src, cmd, test
			if name == module.TestDir && c.Mode != ModeTest {
				break
			}
			// The only Klar project directories that contain buildable modules
			c.Info("Resolving modules in", slog.String("path", fullPath))
			n, err := c.moduleFromDir(name, fullPath, modules, 0, info)
			klarFiles += n
			if err != nil {
				return klarFiles, err
			}
		default:
			// Other directory: ignore
			c.Info("Ignoring directory", slog.String("dir", fullPath))
			continue
		}
	}
	return klarFiles, nil
}
