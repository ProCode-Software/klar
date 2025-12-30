package build

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/build/logger"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
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
	Errors              []errors.CompileError
	Options             []*Options // Configurations from klar.build or CLI
	PreBuild, PostBuild []any      // TODO
	Opener              Opener     // Opens files for reading

	inputs  map[*Input]*InputOptions
	Modules []*Module
	// To avoid reparsing the same file. The same individual file and the
	// file's whole module can be inputs to the compiler.
	flatFiles map[string]*ast.Program
	// Map modules back to configurations
	moduleInputs  map[*Module]*InputOptions
	errorPrinter  *printer.Printer
	WarningLevels map[string]uint8 // Severity levels for warnings
	*slog.Logger                   // TODO: use slog
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
	Manifest   *glaspack.Manifest
	SingleFile bool // Whether the input was a single file
	// TODO: typechecked ast
}

type InputOptions struct {
	Modules  []*Module
	Manifest *glaspack.Manifest
	Options  *Options
}

const (
	_ uint8 = iota
	SuppressWarning
	WarningAsError
)

// File opening
// ============

type OpenFile struct {
	Size      int64
	ShortPath string
	io.ReadCloser
}

// Opener opens files for reading. The size parameter is the size of
// the file in bytes. [io.NopCloser] can be used to wrap a [io.Reader]
// if nothing needs to be done when closing.
type Opener interface {
	Open(name string) (*OpenFile, error)
}

// TokenOpener is an [Opener] that can also provide a file's tokens.
type TokenOpener interface {
	Opener
	OpenTokens(file string) (tokens []lexer.Token, shortPath string, err error)
}

// StdOpener implements [Opener] and is the standard implementation that reads
// Klar files on the system.
type StdOpener struct{ cwd string }

// Open implements [Opener]. Open returns the [*os.File], the size of the file in
// bytes when calling [os.File.Stat], and any error that occurred while opening.
func (o StdOpener) Open(name string) (f *OpenFile, err error) {
	fr, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := fr.Stat()
	if err != nil {
		return nil, err
	}
	relPath, err := filepath.Rel(o.cwd, name)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// fallback to absolute path
		relPath = name
	}
	return &OpenFile{
		Size:       stat.Size(),
		ShortPath:  relPath,
		ReadCloser: fr,
	}, nil
}

// A SingleOpener is a [Opener] that opens only one file.
type SingleOpener struct {
	Path, ShortPath string
	Reader          io.ReadCloser
}

// Open reads from o.Reader and returns a nil error if name == o.FileName,
// otherwise it returns [os.ErrNotExist].
func (o *SingleOpener) Open(name string) (f *OpenFile, err error) {
	if name != o.Path {
		return nil, os.ErrNotExist
	}
	var size int64
	// Estimate size if possible
	switch r := o.Reader.(type) {
	case *os.File:
		if stat, err := r.Stat(); err == nil {
			size = stat.Size()
		}
	case interface{ Len() int }:
		size = int64(r.Len())
	}
	return &OpenFile{
		Size:       size,
		ShortPath:  o.ShortPath,
		ReadCloser: o.Reader,
	}, nil
}

// SingleTokenOpener is a [TokenOpener] that opens only one file.
type SingleTokenOpener struct {
	Path, ShortPath string
	Tokens          []lexer.Token
}

// OpenTokens returns (o.Tokens, o.ShortPath, nil) if name == o.Path,
// otherwise it returns [os.ErrNotExist].
func (o *SingleTokenOpener) OpenTokens(name string) ([]lexer.Token, string, error) {
	if name != o.Path {
		return nil, "", os.ErrNotExist
	}
	return o.Tokens, o.ShortPath, nil
}

// Open returns [os.ErrNotExist].
func (o *SingleTokenOpener) Open(string) (*OpenFile, error) {
	return nil, os.ErrNotExist
}

// Logging
// ==========

// CloseLogger closes the logger if it is a [LogHandler] and the output file needs closing.
func (c *Compiler) CloseLogger() error {
	if h, ok := c.Logger.Handler().(*logger.LogHandler); ok {
		return h.Close()
	}
	return nil
}

func (c *Compiler) _logBase(log func(string, ...any), msg string, v ...any) {
	if c.Logger != nil && c.Logger.Handler() != nil {
		if h, ok := c.Logger.Handler().(*logger.LogHandler); ok {
			h.SetSkip(5)
		}
		log(msg, v...)
	}
}

func (c *Compiler) LogInfo(msg string, v ...any)  { c._logBase(c.Logger.Info, msg, v...) }
func (c *Compiler) LogError(msg string, v ...any) { c._logBase(c.Logger.Error, msg, v...) }

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
