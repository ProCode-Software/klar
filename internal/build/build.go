package build

import (
	"io"
	"log/slog"
	"os"
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

	moduleInputs  map[*Module]*InputOptions // Map modules back to configurations
	Reporter      *reporter.Reporter        // Reports errors to the console
	WarningLevels map[string]uint8          // Severity levels for warnings
	*slog.Logger
}

type (
	InputKind int
	BuildMode int
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
}

type InputOptions struct {
	Modules  []*Module
	Manifest *glaspack.Manifest
	PkgInfo  *module.PackageInfo
	Options  *Options
}

const (
	_ uint8 = iota
	SuppressWarning
	WarningAsError
)

// Logging
// ==========

// CloseLogger closes the logger if it is a [LogHandler] and the output file needs closing.
func (c *Compiler) CloseLogger() error {
	if h, ok := c.Logger.Handler().(*logger.LogHandler); ok {
		return h.Close()
	}
	return nil
}

const showFileInLogs = false

// SetLogger sets b's Logger and verbosity. If verbose is true, b.Logger is set
// to [os.Stderr]. If the $KLAR_LOG_FILE environment variable is set, regardless
// of the value of verbose, b.Logger is set to write to that file. Otherwise,
// b.Logger is set to nil. SetLogger returns an error if it fails to
// open $KLAR_LOG_FILE.
func SetLogger(b *Compiler, verbose, json bool) error {
	var (
		logFile = os.Getenv("KLAR_LOG_FILE")
		out     io.Writer
		flags   logger.Flags
	)
	switch {
	case logFile != "":
		file, err := os.Create(logFile)
		if err != nil {
			return &FilesystemError{"create", "KLAR_LOG_FILE", err}
		}
		out = file
		flags |= logger.NoColor
	case verbose:
		out = os.Stderr
	default:
		return nil
	}
	if json {
		b.Logger = slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
			AddSource: showFileInLogs,
		}))
		return nil
	}
	if showFileInLogs {
		flags |= logger.ShowSource
	}
	b.Logger = slog.New(logger.NewLogHandler(out, flags))
	return nil
}
