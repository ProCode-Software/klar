package run

import (
	"io"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func Run(r *command.Runner) {
}

// Errors are already reported to standard error
func RunInput(r io.Reader, fileName string) (*build.BuildResult, error) {
	// Don't need to resolve files
	c, _ := build.NewCompiler(build.ModRun)
	c.Parser = build.NewStaticParser(fileName, &build.StaticParserFile{
		ShortPath: fileName,
		Reader:    r,
	})
	c.AddInputs(build.Input{Kind: build.KindFile, Name: fileName, Path: fileName})
	return compile(c)
}

// Errors are already reported to standard error
func RunTokens(tokens []lexer.Token, fileName string) (*build.BuildResult, error) {
	// Don't need to resolve files
	c, _ := build.NewCompiler(build.ModRun)
	c.Parser = build.NewStaticParser(fileName, &build.StaticParserFile{
		ShortPath: fileName,
		Tokens:    tokens,
	})
	c.AddInputs(build.Input{Kind: build.KindFile, Name: fileName, Path: fileName})
	return compile(c)
}

func compile(c *build.Compiler) (*build.BuildResult, error) {
	c.StartTime = time.Now()
	res, err := c.Compile()
	c.PrintAllErrors(res.Errors)
	if err != nil {
		if ie, ok := err.(*build.InterfaceError); ok {
			build.PrintInterfaceErr(ie)
		} else {
			cli.Error(err.Error())
		}
	}
	if len(res.Errors) > 0 {
		err = res.Errors[0]
	}
	return res, err
}

const LongDescription = ``
