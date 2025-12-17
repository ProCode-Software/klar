package build

import (
	"time"

	"github.com/ProCode-Software/klar/internal/errors"
)

type BuildResult struct {
	Errors []errors.CompileError
	// Time from [Compiler.StartTime] to finish time.
	Elapsed time.Duration
	// Whether the build stopped early due to too many errors
	EarlyExit bool
}

func (c *Compiler) Compile() (res *BuildResult, err error) {
	res = &BuildResult{}
	defer func() { res.Elapsed = time.Since(c.StartTime) }()
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
		parseErrs = nil
		c.LogError("Build failed due to errors")
		return
	}
	return
}
