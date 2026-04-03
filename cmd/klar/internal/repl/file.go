package repl

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/lexer"
	astParser "github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/ProCode-Software/klar/pkg/printer"
)

func parseArg(tokens []lexer.Token) (string, bool) {
	if len(tokens) < 1 {
		return "", false
	}
	if tokens[0].Kind == lexer.String {
		p := astParser.Parser{Tokens: tokens}
		str := p.ParseString()
		return str.Content, true
	}
	return string(printer.PrintTokens(tokens)), true
}

func (s *Session) fileError(file, op string, err error) {
	if errors.Is(err, fs.ErrNotExist) {
		s.Printf(ansi.CodeBrightRed, "File not found: %s", ansi.BrightCyan(file))
		return
	}
	s.Printf(ansi.CodeBrightRed, "Failed to %s to %s: %v", op, ansi.BrightCyan(file), err)
}

func (s *Session) LoadFile(args []lexer.Token) {
	path, ok := parseArg(args)
	if !ok {
		s.Printf(ansi.CodeBrightRed, "Missing file path. Usage: %s %s",
			ansi.Yellow("load"), ansi.Cyan("<file>"))
		return
	}
	f, err := os.Open(path)
	// Implicit file extension
	if errors.Is(err, fs.ErrNotExist) && !strings.HasSuffix(path, ".klar") {
		f, err = os.Open(path + ".klar")
	}
	if err != nil {
		s.fileError(path, "open", err)
		return
	}
	defer f.Close()
	tokens, err := parser.TokenizeFile(f)
	if err != nil {
		s.handleLexerError(err)
		return
	}
	s.runTokens(tokens)
}

func (s *Session) SaveFile(args []lexer.Token) {
	path, ok := parseArg(args)
	if !ok {
		// A file path only has to be provided once in a session
		if path = s.lastSaveLoc; path == "" {
			if args == nil {
				// Keyboard shortcut used
				s.Printf(ansi.CodeBrightRed,
					"Please provide a file path by manually typing %s",
					ansi.Cyan("save <file>"),
				)
			} else {
				// Command line arguments used
				s.Printf(ansi.CodeBrightRed,
					"Please provide a file path after the %s command",
					ansi.Cyan("save"),
				)
			}
			return
		}
	} else {
		s.lastSaveLoc = path
	}
	file, err := os.Create(path)
	if err != nil {
		s.fileError(path, "write to", err)
		return
	}
	defer file.Close()
	// Write successfully evaluated lines to the file
	for _, tok := range s.evaluated {
		fmt.Fprintf(file, "%s\n", printer.PrintTokens(tok))
	}
}
