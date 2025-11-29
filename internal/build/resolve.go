package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/module"
)

const sep = string(filepath.Separator)

// Step 1: Determine the kind of each input and resolve its klar.build file
// =====

func ResolveInputs(inputs []string) ([]Input, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	res := make([]Input, 0, len(inputs))
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
			i = Input{Kind: KindModule, Name: input[1:]}
			// TODO: resolve klar.build
		default:
			info, err := os.Stat(input)
			if err != nil {
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
				i = Input{Path: fullPath, Name: filepath.Base(fullPath)}
				if isPkg, err := module.IsPackage(fullPath); err != nil {
					return nil, &FilesystemError{"get", "working directory", err}
				} else if isPkg {
					i.Kind = KindPackage
				} else {
					i.Kind = KindModule
				}
			}
			// Get path to closest klar.build file
			ResolveKlarBuild(&i)
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
	// Look inside directory before outside
	klarBuild := dir + sep + module.BuildFileName
	if _, err := os.Stat(klarBuild); err == nil {
		i.KlarBuild = klarBuild
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
		klarBuild := dir + sep + module.BuildFileName
		if _, err := os.Stat(klarBuild); err == nil {
			i.KlarBuild = klarBuild
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
	c.modules = make(map[string]*Module, len(c.Options))
	// Show an error if no Klar files to compile were found
	checkFileCount := func(klarFiles int, path string, err *error) {
		totalFiles += klarFiles
		if klarFiles == 0 && *err == nil {
			*err = &InterfaceError{Code: ErrNoKlarFiles, Value: path}
		}
	}
	for _, opt := range c.Options {
		for i := range opt.Inputs {
			inp := &opt.Inputs[i]
			info := &InputOptions{}
			// TODO: resolve glas.pack for module/package
			switch inp.Kind {
			case KindPackage:
				c.Log("Resolving package", inp.Path)
				klarFiles, err := c.resolvePackage(inp.Path, &info.Modules, false)
				checkFileCount(klarFiles, inp.Path, &err)
				if err != nil {
					return totalFiles, err
				}
			case KindFile:
				info.Modules = []*Module{{Files: []string{inp.Path}}}
				c.modules[inp.Path] = info.Modules[0]
				totalFiles++
				c.Log("Resolved file", inp.Path)
			case KindStdin:
				// Empty paths are stdin
				info.Modules = []*Module{{Files: []string{""}}}
				c.modules[os.Stdin.Name()] = info.Modules[0]
				totalFiles++
				c.Log("Resolved file from stdin")
			case KindModule:
				c.Log("Resolving module", inp.Path)
				klarFiles, err := c.moduleFromDir(inp.Path, &info.Modules, 0)
				checkFileCount(klarFiles, inp.Path, &err)
				if err != nil {
					return totalFiles, err
				}
			}
			c.inputs[inp] = info
		}
	}
	return totalFiles, nil
}

// moduleFromDir reads the contents of dir, assigning dir to a [*File] with
// dir's contents groupd by submodules (directories), Klar files, and assets
// (non-Klar files). moduleFromDir reports an error if it encounters a
// submodule when depth is [MaxModuleDepth], per the Klar Project Structure spec.
func (c *Compiler) moduleFromDir(dir string, modules *[]*Module, depth int) (
	klarFiles int, err error,
) {
	m := &Module{}
	c.modules[dir], *modules = m, append(*modules, m)

	items, err := os.ReadDir(dir)
	if err != nil {
		return klarFiles, &FilesystemError{"read", dir, err}
	}
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
			c.Log("Resolving submodule:", path)
			more, err := c.moduleFromDir(path, modules, depth+1)
			klarFiles += more
			if err != nil {
				return klarFiles, err
			}
		case IsKlarFile(name):
			m.Files = append(m.Files, path)
			klarFiles++
		default:
			m.Assets = append(m.Assets, path)
		}
	}
	return
}

func (c *Compiler) resolvePackage(path string, modules *[]*Module, nesting bool) (
	klarFiles int, err error,
) {
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
				c.LogError("Found a Klar file in package root:", fullPath)
				err = &InterfaceError{Value: fullPath, Code: ErrFileInPackage}
				return
			}
			continue
		}
		if nesting {
			switch name {
			case module.PackageFolder, "shared", ".klar":
				c.LogError("Found invalid nested project folder", fullPath)
				err = &InterfaceError{Value: fullPath, Code: ErrNestedKlarFolder}
				return
			}
		}
		switch name {
		case module.PackageFolder: // pkg
			pkgs, err := os.ReadDir(fullPath)
			if err != nil {
				c.LogError("Read directory", fullPath, "failed:", err)
				return klarFiles, &FilesystemError{"read", fullPath, err}
			}
			for _, pkg := range pkgs {
				fullPkg := fullPath + sep + pkg.Name()
				if !pkg.IsDir() {
					c.LogErrorf("Found file %s in the %s directory",
						fullPkg, module.PackageFolder,
					)
					return klarFiles,
						&InterfaceError{Code: ErrFileInPkgDir, Value: fullPkg}
				}
				more, err := c.resolvePackage(fullPkg, modules, true)
				klarFiles += more
				if err != nil {
					c.LogError("Resolving subpackage", fullPkg, "failed:", err)
					return klarFiles, err
				}
			}
		case "src", "cmd", "shared", "generated":
			// The only Klar project directories that contain buildable modules
			c.Log("Resolving modules in", fullPath)
			more, err := c.moduleFromDir(fullPath, modules, 0)
			klarFiles += more
			if err != nil {
				return klarFiles, err
			}
		default:
			// Other directory: ignore
			c.Log("Ignoring directory", fullPath)
			continue
		}
	}
	return klarFiles, nil
}
