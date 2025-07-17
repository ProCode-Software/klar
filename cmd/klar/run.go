package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/paths"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/pkg/analysis"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/sanity-io/litter"
)

const INCLUDE_COMMENTS = true

var (
	File        string
	rootProgram ast.Program
	double, _   = target.FromCurrent()
)

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return
	}
	// Is pipe
	tokens, err := parser.TokenizeFile(os.Stdin, INCLUDE_COMMENTS)
	File = "stdin"
	handleErr(err)
	runTokens(tokens)
	os.Exit(0)
}

var errPrinter = errors.Printer{
	Color:     true,
	MaxLines:  3,
	IsRuntime: false,
}

func throw(err error) {
	if !parser.IsKlarError(err) {
		cli.InternalError(err)
	}
	errPrinter.PrintError(err.(errors.KlarError))
}

func runTokens(tokens []lexer.Token) {
	errPrinter.LoadTokens(tokens)
	program, parseErrs := parser.Parse(tokens, &parser.Options{
		File: File,
	})
	rootProgram.Body = append(rootProgram.Body, program.Body...)
	if len(parseErrs) > 0 {
		for i, err := range parseErrs {
			if i != 0 {
				fmt.Println()
			}
			throw(err)
		}
	} else {
		litter.Config.StripPackageNames = true
		// litter.Dump(program)
	}

	// Typecheck
	_, typeErrs := analysis.CheckProgram(rootProgram, analysis.CheckOptions{
		FilePath: File,
		Target:   double,
	})
	if len(typeErrs) > 0 {
		for i, err := range typeErrs {
			if i != 0 {
				fmt.Println()
			}
			throw(err)
		}
	} else {
		fmt.Println(ansi.BoldGreen("✅ No type errors found!"))
	}
}

func RunFile(path string) {
	file, err := os.Open(path)
	if os.IsNotExist(err) && !strings.HasSuffix(strings.ToLower(path), ".klar") {
		path += ".klar"
		file, err = os.Open(path)
	}
	File = paths.Full(path)
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

func RunString(program string) {
	if program == "" {
		return
	}
	tokens, err := parser.TokenizeString(program, INCLUDE_COMMENTS)
	handleErr(err)
	runTokens(tokens)
}
