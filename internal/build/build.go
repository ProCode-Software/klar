package build

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/config/glaspack"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
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
	modules []*Module
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
	KindDir InputKind = 1 << iota
	KindFile
	KindPackage = KindDir | (1 << iota)
	KindModule  = KindDir | (1 << iota)
	KindStdin   = KindFile | (1 << iota)
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
	Modules []*Module
	Project *module.ProjectInfo
	Options *Options
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
	relPath := name
	if !filepath.IsAbs(name) {
		relPath, err = filepath.Rel(o.cwd, name)
		if err != nil || strings.HasPrefix(relPath, "..") {
			// fallback to absolute path
			relPath = name
		}
	}
	return &OpenFile{
		Size:       stat.Size(),
		ShortPath:  relPath,
		ReadCloser: fr,
	}, nil
}

// Logging
// ==========

// CloseLogger closes the logger if it is a [LogHandler] and the output file needs closing.
func (c *Compiler) CloseLogger() error {
	if h, ok := c.Logger.Handler().(*LogHandler); ok {
		return h.Close()
	}
	return nil
}

func (c *Compiler) _logBase(log func(string, ...any), msg string, v ...any) {
	if c.Logger.Handler() != nil {
		log(msg, v...)
	}
}

func (c *Compiler) _logfBase(log func(string, ...any), s string, v ...any) {
	if c.Logger.Handler() != nil {
		log(fmt.Sprintf(s, v...))
	}
}

func (c *Compiler) LogInfo(msg string, v ...any)  { c._logBase(c.Logger.Info, msg, v...) }
func (c *Compiler) LogInfof(s string, v ...any)   { c._logfBase(c.Logger.Info, s, v...) }
func (c *Compiler) LogErrorf(s string, v ...any)  { c._logfBase(c.Logger.Error, s, v...) }
func (c *Compiler) LogError(msg string, v ...any) { c._logBase(c.Logger.Error, msg, v...) }
