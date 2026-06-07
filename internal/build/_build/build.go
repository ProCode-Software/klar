package build

import (
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/build/logger"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/pkg/klarerrors/reporter"
)

// A Compiler compiles Inputs into files.
// The build process consists of the following phases:
//  1. Module resolution: resolves [Input]s into their corresponding [Module]s.
//  2. Input parsing: parses each file in each module into an [ast.Program].
//  3. Type checking & analysis: Performs imports and type-checks each [Module]
//  4. Optimization & IR generation
//  5. Code generation
type Compiler struct {
	Mode                BuildMode
	StartTime           time.Time
	Errors              []*klarerrs.Error
	Options             []*Options // Configurations from klar.build or CLI
	PreBuild, PostBuild []any      // TODO
	Parser              Parser     // Parses files
	WorkDir             string

	inputs  map[*Input]*InputOptions
	Modules []*Module
	// To avoid reparsing the same file. The same individual file and the
	// file's whole module can be inputs to the compiler.
	flatFiles map[string]*ast.Program
	moduleMap moduleMap

	moduleInputs  map[*Module]*InputOptions // Map modules back to configurations
	Reporter      *reporter.Reporter        // Reports errors to the console
	WarningLevels map[string]WarnLevel      // Severity levels for warnings
	*slog.Logger
}

type (
	InputKind int
	BuildMode int
	WarnLevel uint8
)

const (
	ModeBuild   BuildMode = iota // Full compilation
	ModRun                       // Build to cache only
	ModeAnalyze                  // Typed AST only: test, typecheck, LSP
	ModeParse                    // Untyped + resolved AST: format
	ModeTest                     // Resolve test files
)

const (
	KindFile InputKind = iota
	KindPackage
	KindModule
	KindStdin
)

const (
	_ WarnLevel = iota
	SuppressWarning
	WarningAsError
)

type Options struct {
	Inputs []Input
	klarbuild.File
}

type Input struct {
	Kind      InputKind
	Path      string // Filesystem path
	Name      string // Module or package name
	KlarBuild string // Path to klar.build file
}

type File struct {
	Path   string
	Tokens []lexer.Token
	AST    *ast.Program
}

type Module struct {
	Submodules []string                // Submodule paths TODO: needed?
	Files      []string                // Klar file paths. Empty string = stdin
	Assets     []string                // Non-Klar file paths
	Name, Path string                  // Module name and folder/file path
	Programs   map[string]*ast.Program // Base name of files
	SingleFile bool                    // Whether the input was a single file
	Checked    *analysis.Module
	Ready      chan struct{}
}

type InputOptions struct {
	Modules  []*Module
	Manifest *glaspack.Manifest
	PkgInfo  *module.PackageInfo
	Options  *Options
}

type moduleMap struct {
	mu      sync.RWMutex
	modules map[string]*Module
}

// Get returns the module with the given directory path. Get is thread-safe.
func (mm *moduleMap) Get(path string) *Module {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.modules[path]
}

// insertSafe inserts m into the map if it doesn't already exist. It returns
// the existing module if found, or m otherwise. insertSafe is thread-safe.
func (mm *moduleMap) insertSafe(path string, m *Module) *Module {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	if mm.modules == nil {
		mm.modules = make(map[string]*Module)
	}
	if existing := mm.modules[path]; existing != nil {
		return existing
	}
	mm.modules[path] = m
	return m
}

// insert inserts m into the map. Not thread-safe
func (mm *moduleMap) insert(path string, m *Module) {
	if mm.modules == nil {
		mm.modules = make(map[string]*Module)
	}
	mm.modules[path] = m
}
