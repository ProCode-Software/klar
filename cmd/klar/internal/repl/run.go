package repl

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
	"github.com/ProCode-Software/klar/pkg/analysis"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/ergochat/readline"
	"github.com/sanity-io/litter"
)

var ErrPrinter = printer.Printer{MaxLines: 3, Color: true}

var ctrlCMessage = fmt.Sprintf(
	"%[1]sTo exit, type %[2]s'exit'%[1]s, press %[2]sCtrl+D%[1]s, or press %[2]sCtrl+C%[1]s again.",
	ansi.Partial(ansi.CodeYellow), ansi.Partial(ansi.CodeCyan),
)

var (
	defaultPrompt    = ansi.Magenta("> ")
	incompletePrompt = ansi.Green("... ")
)

func Run(*command.Runner) {
	fmt.Println(ansi.Bold("Welcome to Klar"), ansi.Gray("v"+version.KlarVersion))
	fmt.Printf("%[1]sType %[1]shelp%[2]s for more information."+
		"Press %[1]sCtrl+D%[2]s or type %[1]sexit%[2]s to exit.\n",
		ansi.Partial(ansi.CodeReset), ansi.Partial(ansi.CodeCyan),
	)
	rl, err := readline.NewFromConfig(&readline.Config{
		Prompt:          defaultPrompt,
		HistoryFile:     "", // TODO: history file
		InterruptPrompt: ansi.Red("Ctrl+C"),
		EOFPrompt:       ansi.Red("Ctrl+D"),
	})
	if err != nil {
		cli.InternalError(err)
	}
	defer rl.Close()
	log.SetOutput(rl.Stderr())
	var (
		interruptCount   int
		incompleteTokens []lexer.Token
	)
loop:
	for {
		input, err := rl.ReadLine()
		switch err {
		case nil:
		case readline.ErrInterrupt:
			/* if len(input) > 0 { // Never true because of the package
				break // ignore Ctrl+C if there was input
			} */
			if interruptCount == 1 {
				break loop
			}
			fmt.Println(ctrlCMessage)
			interruptCount++
			continue loop
		case io.EOF:
			break loop
		default:
			cli.InternalError(err)
		}
		interruptCount = 0
		tokens, err := parser.TokenizeString(input, true)
		if err != nil { // TODO: maybe better handling
			cli.Error("Lexer error: ", err)
			continue
		}
		if len(tokens) == 2 && tokens[0].Kind == lexer.Identifier {
			switch tokens[0].Source {
			case "exit":
				break loop
			}
		}
		if incompleteTokens != nil {
			tokens = updateIncompleteTokens(incompleteTokens, tokens)
		}
		if isIncomplete(tokens) {
			rl.SetPrompt(incompletePrompt)
			incompleteTokens = tokens
			continue
		} else {
			rl.SetPrompt(defaultPrompt)
			incompleteTokens = nil
		}
		ErrPrinter.LoadTokens(tokens)
		prog, errs := parser.Parse(tokens, &parser.Options{
			File: "repl",
		})
		if len(errs) > 0 {
			printErrors(errs)
			continue
		}
		litter.Dump(prog)
		_, typeErrs := analysis.CheckProgram(prog, analysis.CheckOptions{
			FilePath: "repl",
			Target:   target.Double{Target: target.KlarVM},
		})
		if len(typeErrs) > 0 {
			printErrors(typeErrs)
			continue
		}
	}
}

func printErrors[T errors.KlarError](errs []T) {
	for i, err := range errs {
		if i > 0 {
			fmt.Fprintln(os.Stderr)
		}
		ErrPrinter.PrintError(err)
	}
}

func isIncomplete(tokens []lexer.Token) bool {
	var brackCount int
	for _, tok := range tokens {
		switch tok.Kind {
		case lexer.LeftBracket, lexer.LeftCurlyBrace, lexer.LeftParenthesis,
			lexer.HashLeftCurlyBrace:
			brackCount++
		case lexer.RightBracket, lexer.RightParenthesis, lexer.RightCurlyBrace:
			brackCount--
		}
	}
	return brackCount > 0
}

func updateIncompleteTokens(incompleteTokens, tokens []lexer.Token) []lexer.Token {
	last := len(incompleteTokens)-1
	// Replace EOF with newline
	incompleteTokens[last].Kind = lexer.Newline
	incompleteTokens[last].Source = "\n"
	// Get last line
	lastLine := incompleteTokens[last].Line
	// Update lines of new tokens
	for i := range tokens {
		tokens[i].Line += lastLine
	}
	tokens = append(incompleteTokens, tokens...) // Append new tokens
	return tokens
}
