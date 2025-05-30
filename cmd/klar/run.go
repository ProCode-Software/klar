package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/sanity-io/litter"
)

const INCLUDE_COMMENTS = true

var File string

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return
	}
	// Is pipe
	tokens, err := parser.TokenizeFile(os.Stdin, INCLUDE_COMMENTS)
	File = "stdin"
	handleError(err)
	runTokens(tokens)
	os.Exit(0)
}

var printOptions = errors.PrintOptions{
	Color:    true,
	MaxLines: 5,
	Semantic: true,
}

func stack(err errors.KlarError) string {
	var (
		colon = cli.Color(cli.ANSIDim, ":")
		file  = cli.Color(cli.ANSICyan, File)
		pos   = err.At()
		num   = func(n int) string {
			return cli.Color(cli.ANSIYellow, fmt.Sprint(n))
		}
	)
	return fmt.Sprint("\n    " +
		cli.Color(cli.ANSIDim, "File: ") + file +
		colon + num(pos.Line) +
		colon + num(pos.Col),
	)
}

func throw(err error) {
	if !parser.IsKlarError(err) {
		panic(err)
	}
	var (
		arr     = strings.SplitAfterN(err.Error(), ": ", 3)
		first   = arr[0]
		klarErr = err.(errors.KlarError)
		stack   = stack(klarErr)
	)
	errors.PrintError(klarErr, printOptions)
	if len(arr) < 2 {
		cli.Fail(first, stack)
	}
	errName := strings.TrimSuffix(first, ": ")
	if len(arr) < 3 {
		cli.CustomFailure(errName, arr[1], stack)
	} else {
		cli.CustomFailure(errName, arr[1], arr[2], stack)
	}

}

func runTokens(tokens []lexer.Token) {
	printOptions.Tokens = tokens
	p := parser.NewParser(tokens, parser.ParseOptions{
		ContinueOnError: false,
	})
	program := p.Parse()
	if len(p.Errors) > 0 {
		throw(p.Errors[0])
	}
	litter.Config.StripPackageNames = true
	litter.Dump(program)
}

func RunFile(path string) {
	File = path
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		file, err = os.Open(path + ".klar")
	}
	if err != nil {
		if os.IsNotExist(err) {
			cli.FileNotFound(path)
		}
		cli.InternalError(err)
	}
	tokens, err := parser.TokenizeFile(file, INCLUDE_COMMENTS)
	if err != nil {
		cli.InternalError(err)
	}
	runTokens(tokens)
}

func handleError(err error) {
	if err != nil {
		cli.InternalError(err)
	}
}

func RunString(program string) {
	if File != "<repl>" {
		File = "<string>"
	}
	tokens, err := parser.TokenizeString(program, INCLUDE_COMMENTS)
	handleError(err)
	runTokens(tokens)
}
