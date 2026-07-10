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
	Targets []target.Target
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
	Programs     map[string]*ast.Program // Files in the module that is being checked.
	Errors       []*klarerrs.Error       // Errors reported while type checking.
	Info         *Info
	Options      *Options // Options for type checking.
	module       *Module
	nodeContexts map[ast.Node]*Context // Node where each context begins, excluding top-level

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

func (f FileID) TopLevel() bool { return f == 0 }
func (f FileID) Builtin() bool  { return f < 0 }

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
	c.Info = &Info{
		Expressions: make(map[ast.Expression]*Expr),
	}
	mod.Info = c.Info
	c.module = mod
	c.Programs = mod.Programs
	c.Options = opts
	c.loadInternalModules()
}

func (c *Checker) Check() {
	defer handlePanic()

	sortedFiles := slices.Sorted(maps.Keys(c.Programs))
	// Initialize contexts for each file
	fileContexts := c.initFileContexts(sortedFiles)
	// Perform imports
	c.performImports(sortedFiles, fileContexts)

	// Collect top-level objects in each file and put them in the module
	methods, inits := c.collectTopLevelObjects(sortedFiles, fileContexts)

	// If we're currently bootstrapping, wrap the declared types to allow
	// special operations on them. This must be queued before function bodies.
	if c.module.Flags.Has(BootstrapModule) {
		c.queue(c.wrapBootstrapTypes, false)
	}

	// Check for direct cycles among those objects
	c.checkDirectCycles(c.module.Context)
	// Typecheck those declarations, but not function bodies
	c.checkContextDecls(c.module.Context, methods, inits)

	// Run delayed actions, including checking function bodies & top-level statements
	c.runDelayed(0)

	c.ResetState() // Free memory
}

func (c *Checker) initFileContexts(sortedFiles []string) map[string]*Context {
	fileContexts := make(map[string]*Context, len(sortedFiles))
	c.module.fileID = make(map[FileID]string, len(sortedFiles))
	c.module.fileContext = make(map[FileID]*Context, len(sortedFiles))
	if c.nodeContexts == nil {
		c.nodeContexts = make(map[ast.Node]*Context, len(sortedFiles))
	}
	for i, name := range sortedFiles {
		i := FileID(i) + 1
		c.module.fileID[i] = name
		fileContexts[name] = NewContext(c.module.Context, i)
		c.module.fileContext[i] = fileContexts[name]
	}
	return fileContexts
}

func (c *Checker) CheckedModule() *Module { return c.module }

func (c *Checker) FileContextOf(fid FileID) *Context { return c.module.fileContext[fid] }

// Keeps the created type information
func (c *Checker) ResetState() {
}

func (c *Checker) ResetAll() {
	c.module = nil
	c.Errors = nil
	c.Programs = nil
	c.Options = nil
	c.nodeContexts = nil
}

func (c *Checker) filePath(name string) string {
	if (c.module.Flags & SingleFileModule) != 0 {
		return c.module.Path
	}
	return filepath.Join(c.module.Path, name)
}

func (o *Options) NormalizedTargets(yield func(target.Target) bool) {
	var hasJS bool
	for _, t := range o.Targets {
		if t.IsJavaScript() {
			if hasJS {
				continue
			}
			t = target.JavaScript
			hasJS = true
		}
		if !yield(t) {
			return
		}
	}
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
