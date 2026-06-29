package build

import (
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/util/graph"
	"golang.org/x/sync/errgroup"
)

type Loader struct {
	*Compiler
	*Input
	Deps *Deps
	Root bool
	// If loading modules outside another import, avoid sorting,
	// and add edges to graph instead
	graph *graph.Graph[string]
}

func NewLoader(c *Compiler, i *Input, deps *Deps) *Loader {
	return &Loader{Compiler: c, Input: i, Deps: deps}
}

type Loaded struct {
	// Modules that were loaded from cache and do not need typechecking
	cached []*Module
	// Package dependencies sorted by dependency order. Not all modules may exist
	sortedDeps []string
	// Dependencies that are part of the Klar standard library and need to
	// be compiled first by [PackageCompiler.LoadStdlibDeps]
	stdlibDeps []imports.ImportPath
}

// Load loads the modules of ld's Input as well as their dependencies in the same
// package, parsing their files, and returns a [Loaded] struct.
func (ld *Loader) Load() (*Loaded, error) {
	loaded := &Loaded{}

	// 1. Resolve and parse modules
	modules, _, err := ld.ResolveInputModules()
	if err != nil {
		return nil, err
	}
	var (
		cachedCh         = make(chan *Module, len(modules))
		needsTypeCheckCh = make(chan *Module, len(modules))
		eg               errgroup.Group
		reporterMu       sync.Mutex
	)
	for _, mod := range modules {
		eg.Go(func() error {
			if cached := ld.loadOrParseModule(mod, &eg, &reporterMu); cached {
				cachedCh <- mod
			} else {
				needsTypeCheckCh <- mod
			}
			return nil
		})
	}
	err = eg.Wait()
	close(cachedCh)
	if err != nil {
		close(needsTypeCheckCh)
		return nil, err
	}

	// 2. Delete removed modules/files from cache
	for mod := range cachedCh {
		ld.Deps.Set(mod, mod.Checked.ImportPathString())
		loaded.cached = append(loaded.cached, mod)
		if false {
			// TODO: delete the removed files from mod
			needsTypeCheckCh <- mod
		}
	}
	close(needsTypeCheckCh)

	// If the input is a single file, we're not going to sort the modules.
	if ld.IsSingleFile() {
		if len(needsTypeCheckCh) > 0 {
			mod := <-needsTypeCheckCh
			// Load the stdlib dependencies of the single file. These are the only
			// valid dependencies for a single file.
			// needsTypeCheckCh should have a single module for a single file
			for dep := range mod.Deps {
				if dep.IsStdlib() {
					loaded.stdlibDeps = append(loaded.stdlibDeps, dep)
				}
			}
			// Single-file modules don't have import paths, but we still need a
			// fake one to add to ld.Deps.
			fakeImportPath := mod.Name()
			ld.Deps.Set(mod, fakeImportPath)
			loaded.sortedDeps = []string{fakeImportPath}
		}
		return loaded, nil
	}

	// 3. Order the modules by dependency order
	g := graph.New[string]()
	for mod := range needsTypeCheckCh {
		importPath := ld.PkgInfo.ImportPathOf(mod.Path)
		importPathStr := importPath.String()
		ld.Deps.Set(mod, importPathStr) // Add the module as a dependency

		g.AddVertex(importPathStr)
		for dep := range mod.Deps {
			// 4. Stdlib imports are added to a separate slice to be loaded,
			// unless we're currently loading the stdlib itself. But if the stdlib
			// imports 'klar.js', we want [PackageCompiler.LoadStdlibDeps] to create
			// it instead of [Loader.loadPackageDeps] loading it as a regular module.
			if dep.IsStdlib() && (!importPath.IsStdlib() || (len(dep) > 1 && dep[1] == "js")) {
				loaded.stdlibDeps = append(loaded.stdlibDeps, dep)
				continue // Stdlib modules are always compiled first
			}
			g.AddEdge(dep.String(), importPathStr)
		}
	}
	// 5. Load the dependency modules that are in the current package but not
	// inputs before sorting.
	// Example: If the input is a.b, and a.b depends on a.c, we have to load it.
	if err := ld.loadPackageDeps(g); err != nil {
		return nil, err
	}
	if ld.graph != nil {
		// We are currently doing this right now. Just add the edges
		return nil, nil
	}

	if loaded.sortedDeps, err = g.Toposort(); err != nil {
		return loaded, &InterfaceError{Code: ErrDepCycle, Err: err}
	}
	return loaded, nil
}

// loadPackageDeps loads the dependencies that are in the current package
// but not inputs, and adds them to the graph. If a dependency doesn't
// exist, it is skipped for the typechecker to report an error when imported.
func (ld *Loader) loadPackageDeps(g *graph.Graph[string]) error {
	if len(g.Edges()) == 0 {
		return nil
	}
	inputBase, _, _ := strings.Cut(g.Edges()[0][1], ".")
	for _, edge := range g.Edges() {
		dependency := edge[0]
		if ld.Deps.Has(dependency) {
			continue
		}
		// If 'dependency' is just 'a', base is 'a'
		if base, _, _ := strings.Cut(dependency, "."); base != inputBase &&
			!module.IsPackageDir(base) {
			// Not in the current package. An error will be reported when imported.
			continue
		} /* else if dependency == "klar.js" {
			// We are currently loading an input's stdlib dependencies.
			// (This Loader was created by [PackageCompiler.LoadStdlibDeps])
			// As [PackageCompiler.LoadStdlibDeps] does, don't load 'klar.js'.
			continue
		} */
		modulePath := ld.PkgInfo.ModuleDirOf(imports.NewImportPath(dependency))
		inp := new(*ld.Input) // Shallow copies the input. Added in Go 1.26!
		inp.Path, inp.Kind = modulePath, KindModule

		moduleLoader := NewLoader(ld.Compiler, inp, ld.Deps)
		ld.Root = false
		moduleLoader.graph = g // Don't toposort loaded modules
		if _, err := moduleLoader.Load(); err != nil {
			return err
		}
	}
	return nil
}

func (ld *Loader) loadOrParseModule(m *Module,
	eg *errgroup.Group, reporterMu *sync.Mutex,
) (cached bool) {
	m.ModTimes = make(map[string]time.Time, len(m.Programs))

	// Stdin inputs are never cached
	if ok := ld.Kind != KindStdin && ld.loadFromCache(m); ok {
		return true
	}
	// TODO: Never cache stdin inputs

	// We need to parse each file from scratch
	// ======

	var mu sync.Mutex
	// Sort files for reproducible error outputs
	for _, file := range slices.Sorted(maps.Keys(m.Programs)) {
		if m.Programs[file] != nil {
			// That individual file was unchanged, though some other place in the module
			// was. If that was the case, [Loader.loadFromCache] set this file already.
			continue
		}
		eg.Go(func() error { return ld.parseFile(m, file, reporterMu, &mu) })
		// eg.Wait() will be called by ld.Load()
	}
	return false
}

// loadFromCache attempts to load the module from cache, modifying
// m and returning whether the module is fully cached. loadFromCache may
// set m's individual files and still return false.
func (ld *Loader) loadFromCache(m *Module) (ok bool) {
	// TODO: load from cache
	// Also check that files weren't deleted
	// Also, for each file that wasn't changed, set m.Programs
	// Re-emit warnings
	// Check if stale
	/* var cachedModTime time.Time // TODO: for each program
	for file := range m.Programs {
		path := m.Path
		if !filepath.IsAbs(file) { // Single-file
			path = filepath.Join(m.Path, file)
		}
		stat, err := os.Stat(path)
		if err != nil {
			return false
		}
		if diskModTime := stat.ModTime(); cachedModTime.Before(diskModTime) {
			return false // File changed on disk
		}
	} */
	return false
}
