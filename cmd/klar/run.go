package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/pkg/analysis"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/sanity-io/litter"
)

const INCLUDE_COMMENTS = true

var (
	File        string
	rootProgram ast.Program
)

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
	return fmt.Sprint("    " +
		cli.Color(cli.ANSIDim, "File: ") + file +
		colon + num(pos.Line) +
		colon + num(pos.Col),
	)
}

func throw(anyError error) {
	if !parser.IsKlarError(anyError) {
		panic(anyError)
	}
	var (
		arr   = strings.SplitAfterN(anyError.Error(), ": ", 3)
		first = arr[0]
		err   = anyError.(errors.KlarError)
		stack = stack(err)
	)
	errors.PrintError(err, printOptions)
	if len(arr) < 2 {
		cli.Error(first)
	} else {
		errName := strings.TrimSuffix(first, ": ")
		if len(arr) < 3 {
			cli.CustomError(errName, arr[1])
		} else {
			cli.CustomError(errName, arr[1], arr[2])
		}
	}
	for _, hint := range err.GetHints() {
		cli.HintIndent(hint)
	}
	fmt.Println(stack)
}

func runTokens(tokens []lexer.Token) {
	printOptions.Tokens = tokens
	p := parser.NewParser(tokens, parser.ParseOptions{
		ContinueOnError: false,
	})
	program := p.Parse()
	rootProgram.Body = append(rootProgram.Body, program.Body...)
	if len(p.Errors) > 0 {
		for _, err := range p.Errors {
			throw(err)
		}
	} else {
		litter.Config.StripPackageNames = true
		// litter.Dump(program)
	}

	// Typecheck
	errors := analysis.CheckProgram(rootProgram, analysis.CheckOptions{
		ContinueOnError: true,
	})
	if len(errors) > 0 {
		for _, err := range errors {
			throw(err)
		}
	} else {
		fmt.Println(cli.Color(cli.ANSIGreen+cli.ANSIBold, "✅ No type errors found!"))
	}
}

func RunFile(path string) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		path += ".klar"
		file, err = os.Open(path)
	}
	File = module.ResolvePath(path)
	if err != nil {
		if os.IsNotExist(err) {
			cli.FileNotFound(File)
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
