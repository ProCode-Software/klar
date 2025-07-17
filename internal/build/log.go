package build

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

var (
	infoPrefix = ansi.BoldBlue("info")
	donePrefix = ansi.BoldGreen("done")
	warnPrefix = ansi.BoldYellow("warn")
)

func (c *Compiler) log(prefix, msg string, v ...any) {
	if c.Verbose {
		fmt.Fprintf(os.Stderr, prefix+ansi.BoldDim(": ")+msg+"\n", v...)
	}
}

func (c *Compiler) Log(msg string, v ...any) {
	c.log(infoPrefix, msg, v...)
}

func (c *Compiler) LogDone(msg string, v ...any) {
	c.log(donePrefix, msg, v...)
}
