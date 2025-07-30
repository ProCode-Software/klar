package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
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
	str := ansi.BoldRed(errorType) + ansi.BoldDim(": ") + ansi.Bold(msg)
	if len(detail) > 0 {
		str += fmt.Sprint(detail...)
	}
	fmt.Fprintln(os.Stderr, str)
}

// CustomError prints an error to [os.Stderr] with a custom title.
func CustomError(errorType string, msg string, detail ...any) {
	Custom(errorType, msg, detail...)
}

func CustomErrorStr(msg string) {
	errType := "Error"
	parts := strings.SplitAfterN(msg, ": ", 3)
	var detail string
	if len(parts) > 1 {
		errType = parts[0][:len(parts[0])-2]
		msg = parts[1]
		if len(parts) > 2 {
			detail = parts[2]
		}
	}
	CustomError(errType, msg, detail)
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

func InternalError(err any) {
	Failure("Internal Error: ", err)
}

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
	Custom(ansi.Blue("    Hint"), "", hint)
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
