package build

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/build/cache"
	"github.com/ProCode-Software/klar/internal/klarerrs"
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

	// TODO: If a cached module depends on a module that needs to be
	// recompiled, does the cached module get re-typechecked? Ensure
	// that happens.

	// 2. Delete removed modules/files from cache. TODO: Does
	// this do that?
	for mod := range cachedCh {
		ld.Deps.Set(mod, mod.Checked.ImportPathString())
		loaded.cached = append(loaded.cached, mod)
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
		return loaded, &InterfaceError{Code: ErrImportCycle, Err: err}
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
			if fserr, ok := err.(*FilesystemError); ok && fserr.IsNotExist() &&
				fserr.Path == modulePath {
				// The local dependency doesn't exist in the package
				continue // An error will be reported by the typechecker
			}
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
	if fullyCached := ld.Kind != KindStdin &&
		false; /* ld.loadFromCache(m, reporterMu) */ fullyCached {
		return true
	}

	// Some or all files need to be reparsed and typechecked
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
func (ld *Loader) loadFromCache(m *Module, reporterMu *sync.Mutex) (ok bool) {
	// TODO: We need a custom serialization format for the cache.
	// 1. For [analysis.Module], many objects contain unexported fields,
	// which Gob can't decode, and pointer reference equality isn't preserved.
	// 2. For [ast.Program], all interface nodes have to be pre-registered,
	// so we can't save to cache without doing that.

	ok = true
	// 1. Try to load from cache
	cached, err := cache.Load(ld.PkgInfo.CacheDir(), m.Path)
	if err != nil {
		// TODO: Return the error
		return false
	}
	if cached == nil {
		return false // Not in cache
	}
	for file := range m.Programs {
		path := m.FilePath(file)
		stat, err := os.Stat(path)
		if err != nil {
			ok = false // Idk
			continue
		}
		// 2. Ensure there are no new files
		if _, isNew := cached.Programs[file]; !isNew {
			ok = false
			continue // File created since the module was cached
		}
		// 3. Check mod times of each file. If the file on disk is
		// is newer, we have to parse from scratch.
		cachedModTime := cached.ModTimes[file]
		if diskModTime := stat.ModTime(); cachedModTime.Before(diskModTime) {
			ok = false // File changed on disk
			continue
		}
		// 4. Use the cached AST for unchanged files. If some files in
		// a module changed, the ones that didn't will be set.
		m.Programs[file] = cached.Programs[file]
		m.ModTimes[file] = cached.ModTimes[file]
	}
	// 5. Ensure no files were deleted since the module was cached
	for name := range cached.Programs {
		if _, stillExists := m.Programs[name]; !stillExists {
			ok = false // File was deleted since being cached
		}
	}
	// 6. Re-emit cached warnings
	cachedWarns := cached.Warnings
	if !ok {
		cachedWarns = make([]*klarerrs.Error, 0, len(cached.Warnings))
		for _, warn := range cached.Warnings {
			// Only show warnings from files that are still cached (not
			// changed or deleted)
			if prog, _ := m.Programs[filepath.Base(warn.File)]; prog != nil {
				cachedWarns = append(cachedWarns, warn)
			}
		}
	}
	// Note: The cache doesn't store files' tokens, which are required to display
	// an error's source code. When a warning (or another error that uses one
	// of these files as a detail) needs the tokens, the reporter can lazy-
	// tokenize the file (no parsing needed).
	if hasErrs, _ := ld.sendErrors(cachedWarns); hasErrs {
		panic(fmt.Sprintf("cached module %s shouldn't contain errors", m.Path))
	}
	if ok {
		// 7. If there were no stale files, use the cached typechecked module
		// m.Checked = cached.Checked
		ld.Debug("Module fully loaded from cache", slog.String("path", m.Path))
		return true
	}
	ld.Debug("Module partially loaded from cache", slog.String("path", m.Path))
	return false
}
