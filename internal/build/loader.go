package build

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/graph"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"golang.org/x/sync/errgroup"
)

type Loader struct {
	*Compiler
	*Input
	// StaleModules map[string]*Module // Keys are import paths
	Deps *Deps
}

func NewLoader(c *Compiler, i *Input, deps *Deps) *Loader {
	return &Loader{Compiler: c, Input: i, Deps: deps}
}

type Loaded struct {
	cached     []*Module
	sortedDeps []string // Not all modules may exist
	stdlibDeps []imports.ImportPath
}

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
			if cached := ld.loadOrParseFiles(mod, &eg, &reporterMu); cached {
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

	// 3. Order the modules by dependency order
	g := graph.New[string]()
	for mod := range needsTypeCheckCh {
		importPath := ld.PkgInfo.ImportPathOf(mod.Path)
		importPathStr := importPath.String()
		ld.Deps.Set(mod, importPathStr) // Add the module as a dependency

		g.AddVertex(importPathStr)
		for dep := range mod.Deps {
			// 4. Stdlib imports are added to a separate slice to be loaded
			if dep.IsStdlib() && importPath[0] != "klar" {
				loaded.stdlibDeps = append(loaded.stdlibDeps, dep)
				continue // Stdlib modules are always compiled first
			}
			g.AddEdge(dep.String(), importPathStr)
		}
	}
	if loaded.sortedDeps, err = g.Toposort(); err != nil {
		return loaded, &InterfaceError{Code: ErrDepCycle, Err: err}
	}
	return loaded, nil
}

func (ld *Loader) loadOrParseFiles(m *Module, eg *errgroup.Group,
	reporterMu *sync.Mutex,
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
