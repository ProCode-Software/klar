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
	"github.com/sanity-io/litter"
)

var ErrPrinter = errors.Printer{MaxLines: 3, Color: true}

func Run(*command.Runner) {
	fmt.Printf(
		`%sKlar %s%[5]s
Type %[4]s'help'%[5]s for more information. Press %[4]sCtrl+D%[5]s or %[4]s'exit'%[5]s to exit.
%[3]s`,
		ansi.CodeBold+ansi.CodeYellow, version.KlarVersion, ansi.CodeReset,
		ansi.CodeReset+ansi.CodeCyan, ansi.CodeReset+ansi.CodeDim,
	)
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(ansi.CodeMagenta + "> " + ansi.CodeReset)
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
			for _, err := range errs {
				ErrPrinter.PrintError(err)
			}
			continue
		}
		litter.Dump(prog)
		_, typeErrs := analysis.CheckProgram(prog, analysis.CheckOptions{
			FilePath: "repl",
			Target: target.Double{Target: target.KlarVM},
		})
		if len(typeErrs) > 0 {
			for _, err := range typeErrs {
				ErrPrinter.PrintError(err)
			}
			continue
		}
	}
}
