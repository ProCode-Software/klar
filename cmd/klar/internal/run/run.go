package run

import (
	"io"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func Run(r *command.Runner) {
}

// Runs in its own module
func RunInput(r io.Reader, fileName string) (errs []errors.CompileError, fatalErr error) {
	c := build.NewCompiler(build.ModRun)
	c.AddInputs(build.Input{Kind: build.KindFile, Name: fileName, Path: fileName})
	// Create an opener that reads from r
	var rc io.ReadCloser
	if r, ok := r.(io.ReadCloser); ok {
		rc = r
	} else {
		rc = io.NopCloser(r)
	}
	c.Opener = &build.SingleOpener{fileName, fileName, rc}
	// Compile
	c.StartTime = time.Now()
	res, err := c.Compile()
	return res.Errors, err
}

func RunTokens(tokens []lexer.Token, fileName string) (*build.BuildResult, error) {
	c := build.NewCompiler(build.ModRun)
	c.Opener = &build.SingleTokenOpener{fileName, fileName, tokens}
	c.AddInputs(build.Input{Kind: build.KindFile, Name: fileName, Path: fileName})
	// Compile
	c.StartTime = time.Now()
	return c.Compile()
}

const LongDescription = ``
