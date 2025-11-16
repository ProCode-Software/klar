package build

import (
	"time"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
)

type BuildResult struct {
	Errors  []errors.CompileError
	Elapsed time.Duration
}

func (c *Compiler) Compile() (res *BuildResult, err error) {
	res = &BuildResult{}
	defer func() { res.Elapsed = time.Since(c.StartTime) }()
	if c.ErrorPrinter == nil {
		c.ErrorPrinter = &printer.Printer{MaxLines: 3, Color: true}
	}
	var totalFiles int
	if totalFiles, err = c.ResolveModules(); err != nil {
		return
	}
	var parseErrs []*errors.ParseError
	parseErrs, err = c.ParseModules(totalFiles)
	if err != nil || len(parseErrs) > 0 {
		res.Errors = make([]errors.CompileError, len(parseErrs))
		for i, err := range parseErrs {
			res.Errors[i] = err
		}
		c.LogError("Build failed due to errors")
		return
	}
	return
}
