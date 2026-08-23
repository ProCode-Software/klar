package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/module"
)

func TitlePrefix(color, title string) string {
	return ansi.Color(color, title) + ansi.BoldDim(":") + " "
}

var errorPrefix = TitlePrefix(ansi.CodeBoldBrightRed, "Error")

// CustomError prints an error to [os.Stderr] with a custom title
func CustomError(titleColor, title, msg string, detail ...any) {
	t := strings.TrimSuffix(TitlePrefix(titleColor, title), " ")
	v := []any{t}
	if msg != "" {
		v = []any{t, ansi.Bold(msg)}
	}
	fmt.Fprintln(os.Stderr, append(v, detail...)...)
}

// Error prints an error to [os.Stderr].
func Error(msg string, detail ...any) {
	CustomError(ansi.CodeBoldBrightRed, "Error", msg, detail...)
}

// Warn prints a warning to [os.Stderr].
func Warn(msg string, detail ...any) {
	CustomError(ansi.CodeBoldBrightYellow, "Warning", msg, detail...)
}

func Warnf(msg, detail string, v ...any) {
	f := ansi.Bold(msg)
	if detail != "" {
		f += " " + detail
	}
	fmt.Fprint(
		os.Stderr, TitlePrefix(ansi.CodeBoldBrightYellow, "Warning"),
		fmt.Sprintf(f, v...), "\n",
	)
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
	ansi.TagFprintfln(os.Stderr, "<** r!>Error</r!><dim>:</> "+format, a...)
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
	default:
		panic(r)
	}
}
