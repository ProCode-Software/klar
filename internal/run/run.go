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
	return compile(build.NewStaticParser("", fileName, &build.StaticParserFile{
		ShortPath: fileName,
		Reader:    r,
	}), fileName)
}

// Errors are already reported to standard error
func RunTokens(tokens []lexer.Token, fileName string) (*build.Result, error) {
	// Don't need to resolve files
	return compile(build.NewStaticParser("", fileName, &build.StaticParserFile{
		ShortPath: fileName,
		Tokens:    tokens,
	}), fileName)
}

func compile(parser build.Parser, fileName string) (*build.Result, error) {
	cwd, err := build.Cwd()
	if err != nil {
		return nil, err
	}
	c := build.NewCompiler(build.ModRun, cwd)
	c.Parser = parser
	if p, ok := parser.(*build.StaticParser); ok {
		p.SetFallbackCwd(cwd)
	}

	pc := build.NewProjectCompiler(c)
	pc.Parser = parser
	pc.Inputs = append(pc.Inputs, &build.Input{
		Path: fileName,
		Kind: build.KindFile,
	})

	c.StartTime = time.Now()
	res, err := pc.Compile()
	reportErrors(c, res, err)
	return res, err
}

func reportErrors(c *build.Compiler, res *build.Result, err error) {
	if res != nil {
		// Compile errors
		c.PrintAllErrors(res.Errors)
		return
	}
	// Critical errors
	switch err := err.(type) {
	case nil, *klarerrs.Error:
	case *build.InterfaceError:
		build.PrintInterfaceError(err)
	default:
		cli.Error(err.Error())
	}
}
