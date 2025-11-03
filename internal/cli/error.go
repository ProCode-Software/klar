package cli

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
)

var Colors ansi.Colors

func IsREPL() bool {
	return os.Getenv("KLAR_REPL") == "1"
}

func Print(msg string, detail ...any) {
	Custom("Error", msg, detail...)
}

// Custom prints an error to [os.Stderr] with a custom title
func Custom(errorType string, msg string, detail ...any) {
	str := ansi.BoldBrightRed(errorType) + ansi.BoldDim(": ") + ansi.Bold(msg)
	if len(detail) > 0 {
		str += fmt.Sprint(detail...)
	}
	fmt.Fprintln(os.Stderr, str)
}

// CustomError prints an error to [os.Stderr] with a custom title.
func CustomError(errorType string, msg string, detail ...any) {
	Custom(errorType, msg, detail...)
}

// CustomFailure prints an error to [os.Stderr] with a custom title, followed by a call to
// [os.Exit](1).
func CustomFailure(errorType string, msg string, detail ...any) {
	Custom(errorType, msg, detail...)
	os.Exit(1)
}

// Error prints an error to [os.Stderr].
func Error(msg string, detail ...any) {
	Print(msg, detail...)
}

// Failure prints an error to [os.Stderr], followed by a call to [os.Exit](1).
func Failure(msg string, detail ...any) {
	Print(msg, detail...)
	os.Exit(1)
}

func Failuref(msg, detail string, v ...any) {
	Failure(fmt.Sprintf(ansi.Bold(msg)+detail, v...))
}

func InternalError(err any) {
	Failure("Internal Error: ", err)
}

// TODO: update
func InvalidUsage(title, passed, usage string) {
	Print(title+": ", ansi.Yellow(passed)+"\n\n"+
		ansi.Bold("Usage: ")+ansi.Cyan(usage)+"\n\n"+
		"Use "+ansi.Cyan("'--help'")+" for more information.",
	)
	os.Exit(2)
}

func FileNotFound(path string) {
	Error("File not found: ", path)
	os.Exit(2)
}

func HintIndent(hint string) {
	Custom(ansi.BrightBlue("  Hint"), "", hint)
}

func Hint(hint string) {
	Custom(ansi.BrightBlue("Hint"), "", hint)
}

func HandleInternalErr(err error, detail ...string) {
	if err == nil {
		return
	}
	if len(detail) > 0 {
		InternalError(fmt.Errorf("%s: %w", detail[0], err))
	}
	InternalError(err)
}

func Eprintf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func ErrNoManifest(dir string) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			Failure("Unable to get current working directory: ", err)
		}
		dir = cwd
	}
	Failure("Project not found: ", "Can't find a "+
		ansi.Yellow(module.ManifestName)+" file for "+ansi.Cyan(dir),
	)
}

func ErrNotFound(path, typ string) {
	if typ != "" {
		Failure("Can't find " + typ + " " + ansi.Cyan(path))
	}
	Failure("Can't find " + ansi.Cyan(path))
}
