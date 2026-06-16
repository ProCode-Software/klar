package build

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ProCode-Software/klar/internal/config/glaslock"
	"github.com/ProCode-Software/klar/internal/graph"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/module/imports"
)

type ProjectCompiler struct {
	*Compiler
	Inputs []*Input
	Deps   map[*Input]*Deps
}

type Result struct {
	Modules     []*Module        // Input modules only. In order
	AllModules  map[*Input]*Deps // All modules compiled, including inputs
	DepModules  []*Module        // Excluding inputs. In order
	Elapsed     time.Duration
	Errors      []*klarerrs.Error
	Warnings    []*klarerrs.Error
	IsMaxErrors bool
}

func NewProjectCompiler(c *Compiler) *ProjectCompiler {
	return &ProjectCompiler{Compiler: c}
}

func (pc *ProjectCompiler) ResetState() {
	pc.Compiler.ResetState()
}

func (pc *ProjectCompiler) Compile() (*Result, error) {
	// Compile() may be called multiple times (such as by the LSP)
	pc.ResetState()

	// We need to load the Klar directories before compiling
	if err := module.LoadSystemDirs(); err != nil {
		return nil, err
	}

	// Load the bootstrapped modules that are needed for typechecking
	if err := pc.CompileBootstrapped(); err != nil {
		return nil, err
	}

	// Dependencies are compiled first
	depModules, err := pc.CompileDeps()
	if err != nil {
		return nil, err
	}

	// Don't display errors and warnings from dependencies. When an input imports
	// a dependency with errors, they will have their own error. And for
	// dependency warnings, we don't need them at all.
	pc.ResetErrorsAndWarnings()

	// Then, the inputs from the command line
	inputModules, err := pc.CompileInputs()
	if err != nil {
		return nil, err
	}

	return &Result{
		AllModules: pc.Deps,
		DepModules: depModules,
		Modules:    inputModules,
		Errors:     pc.Errors,
		Warnings:   pc.Warnings,
		Elapsed:    time.Since(pc.StartTime),
	}, nil
}

func (pc *ProjectCompiler) CompileDeps() ([]*Module, error) {
	pc.Deps = make(map[*Input]*Deps, len(pc.Inputs))
	g := graph.NewWithCompare[glaslock.PkgHash](cmp.Compare)
	hashToInput := make(map[glaslock.PkgHash]*Input)
	for _, input := range pc.Inputs {
		lock := input.Lockfile
		if lock == nil {
			continue
		}
		for _, pkg := range lock.Packages {
			if pkg.DevOnly {
				continue // Don't compile dev deps or NPM packages
			}
			hashToInput[pkg.Hash] = input
			g.AddVertex(pkg.Hash)
			for _, dep := range pkg.Deps {
				g.AddEdge(pkg.Hash, dep.Hash)
			}
		}
	}
	sorted, err := g.Toposort()
	if err != nil {
		return nil, &InterfaceError{Code: ErrDepCycle, Err: err}
	}
	modules := make([]*Module, 0, len(sorted))
	for i, hash := range sorted {
		input := hashToInput[hash]
		if input == nil {
			panic(fmt.Sprintf("no input associated with dependency hash %d", hash))
		}
		lockPkg := input.Lockfile.PackageMap[hash]
		dependents := []*Input{input}
		// Find the other inputs that depend on this package. We need to compile for all targets
		for _, inp := range pc.Inputs {
			if inp != input && inp.Lockfile != nil && inp.Lockfile.PackageMap != nil {
				if _, ok := inp.Lockfile.PackageMap[hash]; ok {
					dependents = append(dependents, inp)
				}
			}
		}
		pc.Progress.CompilingDep(lockPkg.Name, i+1, len(sorted))
		// TODO: Run in parallel
		mods, err := pc.CompileDep(dependents, lockPkg)
		if err != nil {
			return nil, err
		}
		modules = append(modules, mods...)
	}
	return modules, nil
}

func (pc *ProjectCompiler) CompileInput(i *Input, root bool) (modules []*Module, err error) {
	pkc := NewPackageCompiler(pc.Compiler, i)
	if pc.Deps[i] == nil {
		pc.Deps[i] = new(make(Deps))
	}
	pkc.Deps = pc.Deps[i]
	pkc.EnforceTargetSupport = root
	return pkc.Compile()
}

func (pc *ProjectCompiler) CompileInputs() (modules []*Module, err error) {
	for _, inp := range pc.Inputs {
		compiled, err := pc.CompileInput(inp, true)
		if err != nil {
			return nil, err
		}
		modules = append(modules, compiled...)
	}
	return modules, nil
}

func (pc *ProjectCompiler) DownloadDeps() error {
	pc.Progress.DownloadingDeps()
	for _, input := range pc.Inputs {
		if input.IsSingleFile() {
			continue
		}
		// TODO: glas install

		// Load the input's lockfile
		lockfilePath := filepath.Join(input.PkgInfo.ProjectDir, module.LockFile)
		if f, err := os.Open(lockfilePath); err == nil {
			defer f.Close()
			if input.Lockfile, err = glaslock.Parse(f); err != nil {
				return fmt.Errorf("failed to parse lockfile at %s: %w", lockfilePath, err)
			}
		}
		// If no lockfile, there are no dependencies
	}
	return nil
}

var isBootstrapping bool

func (pc *ProjectCompiler) CompileBootstrapped() error {
	if isBootstrapping || pc.Mode == ModeParse { // Builtins not needed for parsing
		return nil
	}
	isBootstrapping = true
	defer func() { isBootstrapping = false }()

	importPath := imports.ImportPath{"klar", "_builtin"}
	modulePath := module.SystemDirs.Std + sep + module.SrcDir + sep + filepath.Join(importPath...)
	inp, err := pc.ResolveInput(modulePath, 0)
	if err != nil {
		return err
	}
	compiler := NewPackageCompiler(pc.Compiler, inp)
	compiler.Deps = new(make(Deps))
	compiler.EnforceTargetSupport = false
	if _, err := compiler.Compile(); err != nil {
		return err
	}
	return nil
}

// Inputs that depend on the same package
func (pc *ProjectCompiler) CompileDep(
	inputs []*Input, lockPkg *glaslock.Package,
) (modules []*Module, err error) {
	if lockPkg.From == glaslock.NPM {
		// TODO: load from cache if possible
		// Locate the node_modules folder where the package is installed
		nodeModules := locatePkgNodeModules(lockPkg, inputs)
		if nodeModules == "" {
			return nil, &InterfaceError{
				Code:   ErrDepNotFound,
				Value:  lockPkg.Name,
				Detail: "npm",
			}
		}
		if modules, err = MakeNPMModule(lockPkg, nodeModules); err != nil {
			return nil, err
		}
	} else {
		// Each package may have its own packages folder, but since
		// they will refer to the same package, we're only compiling from one.
		// Though the package will be cached to each of the inputs' cache dir
		root := inputs[0].PkgInfo.PackageDirOf(lockPkg)
		inp, err := pc.ResolveInput(root, 0)
		if err != nil {
			return nil, &InterfaceError{
				Code:  ErrDepNotFound,
				Value: lockPkg.Name,
				Err:   err,
			}
		}
		if modules, err = pc.CompileInput(inp, false); err != nil {
			return nil, err
		}
	}
	// Add the dependency's modules to pc.Deps so each input can import them
	for _, mod := range modules {
		for _, inp := range inputs {
			pc.Deps[inp].Set(mod, mod.Checked.ImportPathString())
		}
	}
	return modules, nil
}

// locatePkgNodeModules returns the path to the node_modules directory that
// contains the given package, or "" if it cannot be found.
func locatePkgNodeModules(pkg *glaslock.Package, inputs []*Input) string {
	for _, inp := range inputs {
		for _, dir := range [...]string{inp.PkgInfo.ProjectDir, inp.PkgInfo.Dir} {
			joined := filepath.Join(dir, "node_modules")
			if _, err := os.Stat(joined); err != nil {
				continue
			}
			// Multiple inputs may depend on their own NPM packages. Ensure this
			// option has this dependency in it.
			if _, err := os.Stat(filepath.Join(joined, pkg.Name)); err == nil {
				return joined
			}
		}
	}
	return ""
}
