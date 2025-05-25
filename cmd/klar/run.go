package main

import (
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/sanity-io/litter"
)

const INCLUDE_COMMENTS = false

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return
	}
	// Is pipe
	tokens, err := parser.TokenizeFile(os.Stdin, INCLUDE_COMMENTS)
	if err != nil {
		cli.InternalError(err)
	}
	runTokens(tokens)
	os.Exit(0)
}

// This is here because Go doesn't let you use []string as []any
func collect(items []string) []any {
	var result []any
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

var printOptions = errors.PrintOptions{
	Color: true,
	MaxLines: 5,
	Semantic: true,
}

func throw(err error) {
	if !parser.IsKlarError(err) {
		panic(err) // Trace needed
		// cli.Fail("Internal Error: ", err)
	}
	arr := strings.SplitAfter(err.Error(), ": ")
	first := arr[0]
	errors.PrintError(err.(errors.KlarError), printOptions)
	if len(arr) < 2 {
		cli.Fail(first)
	}
	errName := first[:len(first)-2]
	cli.CustomFailure(errName, arr[1], collect(arr[2:])...)
}

func runTokens(tokens []lexer.Token) {
	printOptions.Tokens = tokens
	program, errs := parser.ParseTokens(tokens, false)
	if len(errs) > 0 {
		throw(errs[0])
	}
	litter.Config.StripPackageNames = true
	litter.Dump(program)
}

func RunFile(path string) {
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

func RunString(program string) {
	tokens, err := parser.TokenizeString(program, INCLUDE_COMMENTS)
	if err != nil {
		cli.InternalError(err)
	}
	runTokens(tokens)
}
