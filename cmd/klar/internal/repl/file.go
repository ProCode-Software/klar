package repl

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

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
		s.Printf(ansi.CodeRed, "File not found: %s", ansi.Cyan(file))
		return
	}
	s.Printf(ansi.CodeRed, "Failed to %s to %s: %v", op, ansi.Cyan(file), err)
}

func (s *Session) LoadFile(args []lexer.Token) {
	path, ok := parseArg(args)
	if !ok {
		s.Printf(ansi.CodeBrightRed, "Missing file path. Usage: %s %s",
			ansi.Yellow("load"), ansi.Cyan("<file>"))
		return
	}
	f, err := os.Open(path)
	if err != nil {
		s.fileError(path, "open", err)
		return
	}
	defer f.Close()
	tokens, err := parser.TokenizeFile(f, 0)
	if err != nil {
		s.handleLexerError(err)
		return
	}
	s.parse(tokens)
}

func (s *Session) SaveFile(args []lexer.Token) {
	path, ok := parseArg(args)
	if !ok {
		if path = s.lastSaveLoc; path == "" {
			
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
	for _, tok := range s.evaluated {
		fmt.Printf("%#q\n", printer.PrintTokens(tok))
	}
}
