package repl

import (
	"fmt"
	"io"
	"log"

	"github.com/ProCode-Software/klar/cmd/klar/internal/run"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/errors/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
	astParser "github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/pkg/parser"
	"github.com/ergochat/readline"
	"github.com/sanity-io/litter"
)

var (
	defaultPrompt    = ansi.Magenta("> ") // ansi.Magenta("(repl)") + " > "
	incompletePrompt = ansi.Green("... ")

	ErrPrinter = printer.Printer{MaxLines: 3, Color: true}
)

type Session struct {
	tokens      []lexer.Token // Incomplete/multiline tokens
	buf         []byte        // Multiline tokens
	interrupted bool          // If interrupted once
	done        bool
	multiline   bool            // Multiline editing enabled
	line        uint32          // Current line, greater than 0 if multiline
	evaluated   [][]lexer.Token // Successfully evaluated lines
	lastSaveLoc string
	*readline.Instance
}

func NewSession() (*Session, error) {
	hist, err := HistoryFile()
	if err != nil {
		return nil, err
	}
	rl, err := readline.NewFromConfig(&readline.Config{
		Prompt:          defaultPrompt,
		HistoryFile:     hist,
		InterruptPrompt: ansi.Red("Ctrl+C"),
		EOFPrompt:       ansi.Red("exit"),
	})
	if err != nil {
		return nil, err
	}
	log.SetOutput(rl.Stderr())
	return &Session{Instance: rl}, nil
}

func Run(*command.Runner) {
	fmt.Println(ansi.Bold("Welcome to Klar"), ansi.Gray("v"+cli.KlarVersion))
	ansi.ColorPrintln(ansi.CodeGray,
		"Type %s for more information. Press %s or type %s to exit.",
		ansi.Cyan("help"), ansi.Cyan("Ctrl+D"), ansi.Cyan("exit"),
	)
	s, err := NewSession()
	if err != nil {
		cli.InternalError(err)
	}
	for !s.done {
		s.Prompt()
	}
}

func HistoryFile() (string, error) {
	// TODO: history file
	return "", nil
}

// TODO: fix '.' in multiline and incomplete multiline
func (s *Session) Prompt() {
	if s.multiline {
		s.line++
		s.SetPrompt(linePrompt(s.line))
	}
	input, err := s.ReadLine()
	switch err {
	case nil:
	case readline.ErrInterrupt:
		/* if len(input) > 0 { // Never true because of the package
			break // ignore Ctrl+C if there was input
		} */
		if s.interrupted {
			s.Finish()
			return
		}
		fmt.Fprintln(s.Stderr(), ctrlCMessage)
		s.interrupted = true
		return
	case io.EOF:
		s.Finish()
		return
	default:
		cli.InternalError(err)
	}
	s.interrupted = false
	if s.multiline {
		s.buf = append(s.buf, input...)
		s.buf = append(s.buf, '\n')
		s.checkMultilineEnd()
		return
	}
	tokens, err := parser.TokenizeString(input, lexer.IncludeComments)
	if err != nil {
		s.handleLexerError(err)
		return
	}
	if len(tokens) > 1 && tokens[0].Kind == lexer.Identifier {
		valid, exit := s.handleCommand(tokens[0].Source, tokens[1:len(tokens)-1])
		if exit {
			s.Finish()
		}
		if valid {
			return
		}
	}
	s.appendTokens(tokens)
	s.send()
}

func (s *Session) handleLexerError(err error) {
	s.Error("Failed to read tokens", err)
}

func (s *Session) Error(msg string, err error) {
	ansi.Fprintf(s.Stderr(), "<r! **>Error</><**>: %s:</**> %v", msg, err)
}

func (s *Session) send() {
	t := s.tokens
	if isIncomplete(t) {
		s.SetPrompt(incompletePrompt)
		return
	} else {
		s.SetPrompt(defaultPrompt)
		s.tokens = nil
	}
	s.runTokens(t)
}

func (s *Session) runTokens(t []lexer.Token) {
	res, err := run.RunTokens(t, "repl")
	// TODO: get access to typechecked module in order to add Repl flag
	if err != nil {
		return
	}
	litter.Dump(res.Modules[0].Programs["repl"])
	s.evaluated = append(s.evaluated, t)
}

func (s *Session) handleCommand(cmd string, args []lexer.Token) (valid, exit bool) {
	switch cmd {
	case "exit":
		return true, true
	case "help":
		s.PrintHelp()
	case "load":
		s.LoadFile(args)
	case "save":
	case "multiline":
		if s.multiline = !s.multiline; s.multiline {
			fmt.Fprintln(s.Stdout(), multilineEnabledMsg)
		} else {
			fmt.Fprintln(s.Stdout(), multilineDisabledMsg)
			s.sendMultiline()
		}
	default:
		return false, false
	}
	return true, false
}

func isIncompleteToken(tok lexer.TokenType) bool {
	if !astParser.CanEndStatement(tok) {
		switch tok {
		case lexer.Slash, lexer.Newline, lexer.Asterisk:
			return false
		}
		return true
	}
	return false
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
	return brackCount > 0 ||
		(len(tokens) > 1 && isIncompleteToken(tokens[len(tokens)-2].Kind))
}

func (s *Session) appendTokens(newTokens []lexer.Token) {
	if len(s.tokens) == 0 {
		s.tokens = newTokens
		return
	}
	last := len(s.tokens) - 1
	// Replace EOF with newline
	s.tokens[last].Kind = lexer.Newline
	s.tokens[last].Source = "\n"
	// Get last line
	lastLine := s.tokens[last].Line
	// Update lines of new tokens
	for i := range newTokens {
		newTokens[i].Line += lastLine
	}
	s.tokens = append(s.tokens, newTokens...) // Append new tokens
}

func (s *Session) checkMultilineEnd() {
	// Last byte is the newline
	if buf, ln := s.buf, len(s.buf); ln >= 2 && buf[ln-2] == '.' {
		s.buf = buf[:ln-2] // Remove newline and dot
		s.sendMultiline()
	}
}

func (s *Session) sendMultiline() {
	tokens, err := parser.TokenizeBytes(s.buf, lexer.IncludeComments)
	if err != nil {
		s.handleLexerError(err)
		return
	}
	s.tokens = tokens
	s.send()
	s.line = 0
	s.buf = nil
}

func linePrompt(n uint32) string {
	return ansi.Blue(fmt.Sprintf("%2d | ", n))
}

func (s *Session) Printf(color, format string, a ...any) {
	ansi.ColorFprintln(s.Stderr(), color, format, a...)
}

func (s *Session) Oprintf(color, format string, a ...any) {
	ansi.ColorFprintln(s.Stdout(), color, format, a...)
}

func (s *Session) Finish() {
	s.done = true
	s.Close()
}
