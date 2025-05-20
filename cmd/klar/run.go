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
		cli.Fail("Internal Error: ", err)
	}
	tokens, err := parser.TokenizeFile(file, true)
	if err != nil {
		cli.Fail("Internal Error: ", err.Error())
	}
	runTokens(tokens)
}

func RunString(program string) {
	tokens, err := parser.TokenizeString(program, true)
	if err != nil {
		cli.Fail("Internal Error: ", err.Error())
	}
	runTokens(tokens)
}
