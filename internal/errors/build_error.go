package errors

import (
	"fmt"
	"os"
)

func BuildError(err error) {
	fmt.Fprintf(os.Stderr, "\033[1;31m❌ Build failed\033[0;1;2m:\033[0;1m\n    %v\033[m\n", err)
	os.Exit(1)
}

// error from a recover()
func InternalError(err any) {
	BuildError(fmt.Errorf("Internal Error: %v", err))
}