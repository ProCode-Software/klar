package build

import (
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
)

func (c *Compiler) Compile() (
	parseErrs []*errors.ParseError,
	err error,
) {
	if c.ErrorPrinter == nil {
		c.ErrorPrinter = &printer.Printer{MaxLines: 3, Color: true}
	}
	if err = c.ResolveModules(); err != nil {
		return
	}
	parseErrs, err = c.ParseModules()
	if err != nil || len(parseErrs) > 0 {
		c.Errorln("Build failed due to errors")
		return
	}
	return
}
