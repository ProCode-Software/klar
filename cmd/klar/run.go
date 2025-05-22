package main

import (
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/sanity-io/litter"
)

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return
	}
	// Is pipe
	tokens, err := parser.TokenizeFile(os.Stdin, true)
	if err != nil {
		cli.InternalError(err)
	}
	runTokens(tokens)
	os.Exit(0)
}

func runTokens(tokens []lexer.Token) {
	program, errs := parser.ParseTokens(tokens, false)
	if len(errs) > 0 {
		err := errs[len(errs)-1]
		if !parser.IsKlarError(err) {
			cli.Fail("Internal Error: ", err)
		}
		arr := strings.SplitAfter(err.Error(), ": ")
		cli.Fail(arr[0], arr[1])
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
			cli.FileNotFoundError(path)
		}
		cli.InternalError(err)
	}
	tokens, err := parser.TokenizeFile(file, true)
	if err != nil {
		cli.InternalError(err)
	}
	runTokens(tokens)
}

func RunString(program string) {
	tokens, err := parser.TokenizeString(program, true)
	if err != nil {
		cli.InternalError(err)
	}
	runTokens(tokens)
}
