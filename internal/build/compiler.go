package build

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ProCode-Software/klar/internal/build/logger"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/pkg/klarerrors/reporter"
)

type Compiler struct {
	Cwd       string
	Mode      BuildMode
	Reporter  *reporter.Reporter
	StartTime time.Time
	Errors    []*klarerrs.Error
	*slog.Logger
}

func NewCompiler(mode BuildMode, cwd string) *Compiler {
	return &Compiler{
		Mode: mode,
		Cwd:  cwd,
		Reporter: &reporter.Reporter{
			MaxLines:     3,
			Output:       os.Stderr,
			ColorPalette: reporter.DefaultColorPalette(),
			CharacterSet: reporter.DefaultCharacterSet(),
			UseColor:     !ansi.DisableColor,
		},
		Logger: slog.New(slog.DiscardHandler),
	}
}

type BuildMode int

const (
	ModeBuild   BuildMode = iota // Full compilation
	ModRun                       // Build to cache only
	ModeAnalyze                  // Typed AST only: test, typecheck, LSP
	ModeParse                    // Untyped + resolved AST: format
	ModeTest                     // Resolve test files
)

func Cwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", &FilesystemError{"determine", "working directory", err}
	}
	return cwd, nil
}

func (c *Compiler) Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Cwd, path)
}

func (c *Compiler) ResetState() {
	c.Errors = nil
}

// PrintError prints an error to the error printer.
func (c *Compiler) PrintError(err *klarerrs.Error) (int64, error) {
	return c.Reporter.Report(err)
}

func (c *Compiler) PrintAllErrors(errs []*klarerrs.Error) {
	for _, err := range errs {
		c.PrintError(err)
	}
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
