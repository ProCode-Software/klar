package build

import (
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/build/logger"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/pkg/klarerrors/reporter"
)

type Compiler struct {
	WorkDir   string
	Mode      BuildMode
	Reporter  *reporter.Reporter
	StartTime time.Time
	Errors    []*klarerrs.Error
	Warnings  []*klarerrs.Error
	Progress  Progress
	Parser    Parser
	collectMu sync.Mutex
	*slog.Logger
}

func NewCompiler(mode BuildMode, cwd string) *Compiler {
	return &Compiler{
		Mode:    mode,
		WorkDir: cwd,
		Reporter: &reporter.Reporter{
			MaxLines:     3,
			Output:       os.Stderr,
			ColorPalette: reporter.DefaultColorPalette(),
			CharacterSet: reporter.DefaultCharacterSet(),
			UseColor:     !ansi.DisableColor,
		},
		Logger:   slog.New(slog.DiscardHandler),
		Progress: HiddenProgress{},
	}
}

var DefaultStdParserOptions = &parser.Options{MaxErrors: MaxErrors + 1}

func (c *Compiler) UseStdParser() {
	c.Parser = NewStdParser(c.WorkDir, DefaultStdParserOptions)
}

func (c *Compiler) ProgressHidden() bool {
	_, ok := c.Progress.(HiddenProgress)
	return ok
}

type BuildMode int

const (
	ModeBuild   BuildMode = iota // Full compilation
	ModRun                       // Build to cache only
	ModeAnalyze                  // Typed AST only: test, typecheck, LSP
	ModeParse                    // Untyped + resolved AST: format
	ModeTest                     // Resolve test files
)

type Module struct {
	Assets     []string
	Path       string                  // Directory path, or file if single-file
	Programs   map[string]*ast.Program // Keys are file basenames (with extensions)
	ModTimes   map[string]time.Time    // Same basenames as Programs
	Checked    *analysis.Module        // Typechecked module
	SingleFile bool
	Stdin      bool
	Failed     bool // Has errors
}

// Includes the file extension
func (m *Module) FilePath(base string) string {
	if m.SingleFile {
		return m.Path
	}
	return filepath.Join(m.Path, base)
}

// Name returns the module name. If m is a single file, Name returns
// the file name without the extension.
func (m *Module) Name() string {
	if m.Stdin {
		return stdinName
	}
	return strings.TrimSuffix(filepath.Base(m.Path), ".klar")
}

func (m *Module) Deps(yield func(imports.ImportPath) bool) {
	// Sort the file names for reproducible debugging results
	for _, file := range slices.Sorted(maps.Keys(m.Programs)) {
		for dep := range m.Programs[file].Deps {
			if !yield(dep) {
				return
			}
		}
	}
}

type Deps map[string]*Module // Keys are stringed import paths

func (d *Deps) Set(m *Module, importPath string) {
	if *d == nil {
		*d = make(Deps)
	}
	(*d)[importPath] = m
}

func (d *Deps) TryGet(importPath string) (*Module, bool) {
	mod, ok := (*d)[importPath]
	return mod, ok
}

func (d *Deps) Get(importPath string) *Module {
	return (*d)[importPath]
}

func (d *Deps) Has(importPath string) bool {
	_, ok := (*d)[importPath]
	return ok
}

func (c *Compiler) Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.WorkDir, path)
}

func (c *Compiler) ResetState() {
	c.ResetErrorsAndWarnings()
}

func (c *Compiler) ResetErrorsAndWarnings() {
	c.Errors = c.Errors[:0]
	c.Warnings = c.Warnings[:0]
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
