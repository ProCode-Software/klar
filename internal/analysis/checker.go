package analysis

import (
	"maps"
	"path/filepath"
	"slices"
	"sync"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

type Options struct {
	// The [Importer] to use for importing modules. If set to nil, an error is
	// raised when attempting to import a module.
	Importer Importer
	// The target platform for which the program is being compiled.
	// This is needed for resolving platform-specific implementations
	// and modules.
	// TODO: Support multiple targets
	Target target.Target
	// The minimum version of Klar required to compile the program.
	KlarVersion *version.Version
	// If Error != nil, it is called when an error is reported.
	Error func(*klarerrs.Error)
	// Whether the program is being typechecked for testing purposes.
	IsTest bool
	// If MaxErrors > 0, the type checker will terminate when this limit is reached.
	MaxErrors int
	// Whether to skip type checking function bodies.
	IgnoreFuncBodies bool
	// Whether to enforce that objects are supported on all targets in [Options.Target].
	EnforceTargetSupport bool
	// Type checker options from klar.build
	*klarbuild.CheckerOptions
}

type Checker struct {
	Programs    map[string]*ast.Program // Files in the module that is being checked.
	Errors      []*klarerrs.Error       // Errors reported while type checking.
	Info        *Info
	Options     *Options // Options for type checking.
	rootContext *Context // Context where top-level objects are defined.
	module      *Module

	nodeContexts map[ast.Node]*Context
	moduleDecls  map[*Object]*DeclarationInfo // Declaration info for top-level objects

	// For tracking cycles
	objPath      []*Object       // Path of object deps
	objPathIndex map[*Object]int // Indices of objects in objPath

	delayed []action
}

// FileID is the numerical identifier for a [*ast.Program].
//   - FileID <= -1: Builtin context
//   - FileID == 0: Module context
//   - FileID >= 1: File context.
type FileID int

// NewChecker returns an initialized Checker that checks the programs in mod.
// If opts == nil, default options are used.
func NewChecker(mod *Module, opts *Options) *Checker {
	c := &Checker{}
	c.Init(mod, opts)
	return c
}

var DefaultCheckerOptions = &klarbuild.CheckerOptions{
	ValidateExhaustiveness: klarbuild.NoExhaustiveness,
	AllowAssertions:        klarbuild.AllowAssertions,
	CheckedListIndexing:    true,
	CoerceNumbers:          false,
	ValidateExternals:      false,
	CheckAllResults:        false,
	UseAllValues:           false,
}

func (c *Checker) Init(mod *Module, opts *Options) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.CheckerOptions == nil {
		opts.CheckerOptions = DefaultCheckerOptions
	}
	c.rootContext = mod.Context
	c.module = mod
	c.Programs = mod.Programs
	c.Options = opts
	c.moduleDecls = make(map[*Object]*DeclarationInfo)
	c.loadInternalModules()
}

func (c *Checker) Check() {
	defer handlePanic()

	sortedFiles := slices.Sorted(maps.Keys(c.Programs))
	// Initialize contexts for each file
	fileContexts := c.initFileContexts(sortedFiles)
	// Perform imports
	c.performFileImports(sortedFiles, fileContexts)

	// Collect top-level objects in each file and put them in the module
	methods, inits := c.collectTopLevelObjects(sortedFiles, fileContexts)
	// Sort declarations for reproducible error output
	sortedObjs := slices.SortedFunc(maps.Keys(c.moduleDecls), sortByOrder)

	// If we're currently bootstrapping, wrap the declared types to allow
	// special operations on them.
	if c.module.Flags.Has(BootstrapModule) {
		c.wrapCompositeBootstrapTypes()
	}

	// Check for direct cycles among those objects
	c.checkDirectCycles(sortedObjs)
	// Typecheck those declarations, but not function bodies
	c.checkTopLevelObjects(sortedObjs, methods, inits)

	// Run delayed actions, including checking function bodies
	c.runDelayed(0)

	c.ResetState() // Free memory
}

func (c *Checker) initFileContexts(sortedFiles []string) map[string]*Context {
	fileContexts := make(map[string]*Context, len(sortedFiles))
	c.module.fileID = make(map[FileID]string, len(sortedFiles))
	if c.nodeContexts == nil {
		c.nodeContexts = make(map[ast.Node]*Context, len(sortedFiles))
	}
	for i, name := range sortedFiles {
		i := FileID(i) + 1
		c.module.fileID[i] = name
		fileContexts[name] = NewContext(c.rootContext, i)
	}
	return fileContexts
}

func (c *Checker) CheckedModule() *Module { return c.module }

// Keeps the created type information
func (c *Checker) ResetState() {
}

func (c *Checker) ResetAll() {
	c.module = nil
	c.Errors = nil
	c.rootContext = nil
	c.Programs = nil
	c.Options = nil
	c.moduleDecls = nil
	c.nodeContexts = nil
	c.moduleDecls = nil
}

func (c *Checker) filePath(name string) string {
	if (c.module.Flags & SingleFileModule) != 0 {
		return c.module.Path
	}
	return filepath.Join(c.module.Path, name)
}

func (c *Checker) pushToPath(o *Object) {
	if c.objPathIndex == nil {
		c.objPathIndex = make(map[*Object]int)
	}
	c.objPathIndex[o] = len(c.objPath)
	c.objPath = append(c.objPath, o)
}

func (c *Checker) popPath() {
	i := len(c.objPath) - 1
	delete(c.objPathIndex, c.objPath[i])
	c.objPath[i] = nil
	c.objPath = c.objPath[:i]
}

func (c *Checker) queue(f func(), runInParallel bool) {
	c.delayed = append(c.delayed, action{f, runInParallel})
}

type action struct {
	f        func()
	parallel bool
}

// runDelayed runs the delayed actions pushed after from.
func (c *Checker) runDelayed(from int) {
	var wg sync.WaitGroup
	// Don't use a 'range' loop because delayed functions could push to the stack
	for i := from; i < len(c.delayed); i++ {
		a := c.delayed[i]
		if a.parallel {
			// TODO: they could append, so be careful about races
			// wg.Go(a.f)
			a.f()
		} else {
			a.f()
		}
	}
	wg.Wait()
	c.delayed = c.delayed[:from]
}
