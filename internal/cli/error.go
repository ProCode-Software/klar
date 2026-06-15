package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
)

var errorPrefix = ansi.BoldBrightRed("Error") + ansi.BoldDim(": ")

// Custom prints an error to [os.Stderr] with a custom title
func Custom(errorType string, msg string, detail ...any) {
	str := ansi.BoldBrightRed(errorType) + ansi.BoldDim(":")
	v := []any{str}
	if msg != "" {
		v = []any{str, ansi.Bold(msg)}
	}
	fmt.Fprintln(os.Stderr, append(v, detail...)...)
}

// Error prints an error to [os.Stderr].
func Error(msg string, detail ...any) {
	Custom("Error", msg, detail...)
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

func Failuref(msg, detail string, v ...any) {
	f := errorPrefix + ansi.Bold(msg) + detail + "\n"
	fmt.Fprintf(os.Stderr, f, v...)
}

func InternalError(detail ...any) {
	Failure("Internal Error:", detail...)
}

func HintIndent(hint string) {
	Custom(ansi.BrightBlue("  Hint"), "", hint)
}

func Hint(hint string) {
	Custom(ansi.BrightBlue("Hint"), "", hint)
}

func Eprintf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func ColorErrorfln(format string, a ...any) {
	ansi.Fprintfln(os.Stderr, "<** r!>Error</r!><dim>:</> "+format, a...)
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

func HandleSignalExit() {
	switch r := recover().(type) {
	case SignalExit:
		os.Exit(r.Code)
	case nil:
		return
	default:
		panic(r)
	}
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

	var res string
	fmt.Scanln(&res) // Response
	switch strings.ToLower(strings.TrimSpace(res)) {
	case "":
		return defaultRes
	case "y", "yes", "true":
		return true
	case "n", "no", "false":
		return false
	default:
		return defaultRes
	}
}
