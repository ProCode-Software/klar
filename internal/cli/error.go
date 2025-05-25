package cli

import (
	"fmt"
	"os"
)

func isREPL() bool {
	return os.Getenv("KLAR_REPL") == "1"
}

func Print(msg string, detail ...any) {
	Custom("Error", msg, detail...)
}

// Custom prints an error to [os.Stderr] with a custom title
func Custom(errorType string, msg string, detail ...any) {
	str := ANSIBoldRed + errorType + ANSIResetBoldDim + ": " + ANSIResetBold + msg + ANSIReset
	if detail != nil && len(detail) > 0 {
		str += fmt.Sprintf("%v", detail...)
	}
	fmt.Fprintln(os.Stderr, str)
}

// CustomError prints an error to [os.Stderr] with a custom title, followed by a call to
// [os.Exit](1).
func CustomFailure(errorType string, msg string, detail ...any) {
	Custom(errorType, msg, detail...)
	if !isREPL() {
		os.Exit(1)
	}
}

// Fail prints an error to [os.Stderr], followed by a call to [os.Exit](1).
func Fail(msg string, detail ...any) {
	Print(msg, detail...)
	if !isREPL() {
		os.Exit(1)
	}
}

func InternalError(err any) {
	Fail("Internal Error: ", err)
}

func InvalidUsage(usage string) {
	Print("Invalid usage", "Usage: "+usage)
	os.Exit(2)
}

func FileNotFound(path string) {
	Fail("File not found: ", path)
}
