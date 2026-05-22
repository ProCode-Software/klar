package run

import (
	"io"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func RunString(s, fileName string) (*build.Result, error) {
	return RunInput(strings.NewReader(s), fileName)
}

// Errors are already reported to standard error
func RunInput(r io.Reader, fileName string) (*build.Result, error) {
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
func RunTokens(tokens []lexer.Token, fileName string) (*build.Result, error) {
	// Don't need to resolve files
	c, _ := build.NewCompiler(build.ModRun)
	c.Parser = build.NewStaticParser(fileName, &build.StaticParserFile{
		ShortPath: fileName,
		Tokens:    tokens,
	})
	c.AddInputs(build.Input{Kind: build.KindFile, Name: fileName, Path: fileName})
	return compile(c)
}

func compile(c *build.Compiler) (*build.Result, error) {
	c.StartTime = time.Now()
	res, err := c.Compile()
	reportErrors(c, res, err)
	if err == nil && len(res.Errors) > 0 {
		err = res.Errors[0]
	}
	return res, err
}

func reportErrors(c *build.Compiler, res *build.Result, err error) {
	// Compile errors
	c.PrintAllErrors(res.Errors)
	// Critical errors
	switch err := err.(type) {
	case nil, *klarerrs.Error:
	case *build.InterfaceError:
		build.PrintInterfaceError(err)
	default:
		cli.Error(err.Error())
	}
}
