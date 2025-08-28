package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
	"github.com/ProCode-Software/klar/pkg/analysis"
	"github.com/ProCode-Software/klar/pkg/parser"
	// "github.com/sanity-io/litter"
)

var ErrPrinter = errors.Printer{MaxLines: 3, Color: true}

func Run(*command.Runner) {
	fmt.Println(ansi.Bold("Welcome to Klar"), "v"+version.KlarVersion)
	fmt.Println(
		ansi.Dim("Type"), ansi.Cyan("'help'"), ansi.Dim("for more information. Press"),
		ansi.Cyan("Ctrl+D"), ansi.Dim("or type"), ansi.Cyan("'exit'"), ansi.Dim("to exit."),
	)
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(ansi.Magenta("> "))
		input, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			cli.InternalError(err)
		}
		input = strings.TrimSpace(input)
		if input == "exit" {
			break
		}
		if input == "" {
			continue
		}
		tokens, err := parser.TokenizeString(input, true)
		if err != nil {
			cli.Error("Lexer error: ", err)
			continue
		}
		ErrPrinter.LoadTokens(tokens)
		prog, errs := parser.Parse(tokens, &parser.Options{
			File: "repl",
		})
		if len(errs) > 0 {
			for i, err := range errs {
				if i > 0 {
					fmt.Println()
				}
				ErrPrinter.PrintError(err)
			}
			continue
		}
		//litter.Dump(prog)
		_, typeErrs := analysis.CheckProgram(prog, analysis.CheckOptions{
			FilePath: "repl",
			Target:   target.Double{Target: target.KlarVM},
		})
		if len(typeErrs) > 0 {
			for i, err := range typeErrs {
				if i > 0 {
					fmt.Println()
				}
				ErrPrinter.PrintError(err)
			}
			continue
		}
	}
}
