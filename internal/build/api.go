package build

import (
	"io"
	"log"
	"os"

	"github.com/ProCode-Software/klar/internal/errors"
)

func NewCompiler(mode BuildMode) *Compiler {
	return &Compiler{
		Mode: mode,
	}
}

// UseStdOpener sets c's Opener to the standard opener. UseStdOpener
// returns an error if it fails to get the working directory.
func (c *Compiler) UseStdOpener() error {
	cwd, err := os.Getwd()
	if err != nil {
		return &FilesystemError{"get", "working directory", err}
	}
	c.Opener = StdOpener{cwd: cwd}
	return nil
}

// PrintError prints an error to the error printer.
func (c *Compiler) PrintError(err errors.CompileError) {
	c.errorPrinter.PrintError(err)
}

// SetLogger sets c's Logger and verbosity. If verbose is true, c.Logger is set
// to [os.Stderr]. If the $KLAR_LOG_FILE environment variable is set, regardless
// of the value of verbose, c.Logger is set to write to that file. Otherwise,
// c.Logger is set to [io.Discard]. SetLogger returns an error if it fails to
// open $KLAR_LOG_FILE.
func (c *Compiler) SetLogger(verbose bool) error {
	logFile := os.Getenv("KLAR_LOG_FILE")
	var out io.Writer
	switch {
	case logFile != "":
		file, err := os.Create(logFile)
		if err != nil {
			return &FilesystemError{"create", "KLAR_LOG_FILE", err}
		}
		c.openFiles = append(c.openFiles, file)
		out = file
		c.verbose = true
	case verbose:
		out = os.Stderr
		c.verbose = true
	default:
		out = io.Discard
	}
	c.Logger = log.New(out, "[compiler] ", log.Ltime)
	return nil
}
