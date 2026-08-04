package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
	"golang.org/x/term"
)

var errorPrefix = ansi.BoldBrightRed("Error") + ansi.BoldDim(": ")

// CustomError prints an error to [os.Stderr] with a custom title
func CustomError(errorType string, msg string, detail ...any) {
	title := ansi.BoldBrightRed(errorType) + ansi.BoldDim(":")
	v := []any{title}
	if msg != "" {
		v = []any{title, ansi.Bold(msg)}
	}
	fmt.Fprintln(os.Stderr, append(v, detail...)...)
}

// Error prints an error to [os.Stderr].
func Error(msg string, detail ...any) {
	CustomError("Error", msg, detail...)
}

// Warn prints a warning to [os.Stderr].
func Warn(msg string, detail ...any) {
	CustomError(ansi.BoldBrightYellow("Warning"), msg, detail...)
}

// Failure prints an error to [os.Stderr], followed by a call to [os.Exit](1).
func Failure(msg string, detail ...any) {
	Error(msg, detail...)
	Exit(1)
}

// FailureError is equivalent to [Failure](err.Error())
func FailureError(err error) {
	Failure(err.Error())
}

func Failuref(msg string, v ...any) {
	f := errorPrefix + ansi.Bold(msg) + "\n"
	fmt.Fprintf(os.Stderr, f, v...)
	Exit(1)
}

func FailureDetailf(msg, detail string, v ...any) {
	f := errorPrefix + ansi.Bold(msg) + detail + "\n"
	fmt.Fprintf(os.Stderr, f, v...)
	Exit(1)
}

func InternalError(detail ...any) {
	Failure("Internal Error:", detail...)
}

func HintIndent(hint string) {
	CustomError(ansi.BrightBlue("  Hint"), "", hint)
}

func Hint(hint string) {
	CustomError(ansi.BrightBlue("Hint"), "", hint)
}

func Eprintf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func ColorErrorfln(format string, a ...any) {
	ansi.TagFprintfln(os.Stderr, "<** r!>Error</r!><dim>:</><**> "+format, a...)
}

func ErrNoManifest(dir string) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			Failure("Unable to get current working directory:", err)
		}
		dir = cwd
	}
	Failure(
		"Project not found:", "Can't find a "+
			ansi.Yellow(module.ManifestFile)+" file for "+ansi.Cyan(dir),
	)
}

func ErrNotFound(path, typ string) {
	if typ != "" {
		Error("Can't find " + typ + " " + ansi.Cyan(path))
	} else {
		Error("Can't find " + ansi.Cyan(path))
	}
	Exit(2)
}

type SignalExit struct{ Code int }

// Exit panics with a [SignalExit]. This should be used instead of [os.Exit]
// to ensure deferred functions are run before exiting. This is caught by the
// [HandleSignalExit] and calls [os.Exit] with the provided code.
func Exit(code int) {
	panic(SignalExit{code})
}

func Confirm(msg string, defaultRes bool) bool {
	// (y/n) display
	var defaultStr string
	if defaultRes {
		defaultStr = ansi.ColorSprintf(ansi.CodeDim, "(%s/n)", ansi.BoldBrightGreen("Y"))
	} else {
		defaultStr = ansi.ColorSprintf(ansi.CodeDim, "(y/%s)", ansi.BoldBrightRed("N"))
	}

	fmt.Printf("%s %s: ", msg, defaultStr) // Prompt
	defer fmt.Println()                    // Final newline

	// The terminal has to be made raw so we can read a single character without
	// the user pressing Enter
	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}
	for {
		res := make([]byte, 1)
		os.Stdin.Read(res)
		switch bytes.ToLower(res)[0] {
		case ' ', '\n', '\t', 'r':
			continue
		case 'y', 't', '1':
			return true
		case 'n', 'f', '0':
			return false
		default:
			return defaultRes
		}
	}
}
