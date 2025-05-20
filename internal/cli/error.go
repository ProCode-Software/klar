package cli

import (
	"fmt"
	"os"
)

const (
	ANSIBoldRed      = "\033[1;31m"
	ANSIReset        = "\033[m"
	ANSIBoldDim      = "\033[1;2m"
	ANSIBold         = "\033[1m"
	ANSIDim          = "\033[2m"
	ANSIResetBold    = ANSIReset + ANSIBold
	ANSIResetBoldDim = ANSIReset + ANSIBoldDim
)

func PrintError(msg string, detail any) {
	str := ANSIBoldRed + "Error" + ANSIResetBoldDim + ": " + ANSIResetBold + msg + ANSIReset
	if detail != nil && detail != "" {
		str += fmt.Sprintf("%v", detail)
	}
	fmt.Println(str)
}

func Fail(msg string, detail any) {
	PrintError(msg, detail)
	os.Exit(1)
}

func InvalidUsage(usage string) {
	PrintError("Invalid usage", "Usage: "+usage)
	os.Exit(2)
}

func FileNotFoundError(path string) {
	Fail("File not found: ", path)
}
