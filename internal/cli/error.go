package cli

import (
	"fmt"
	"os"
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

func InternalError(err any) {
	Fail("Internal Error: ", err)
}

func InvalidUsage(usage string) {
	PrintError("Invalid usage", "Usage: "+usage)
	os.Exit(2)
}

func FileNotFoundError(path string) {
	Fail("File not found: ", path)
}
