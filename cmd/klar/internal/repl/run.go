package repl

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module"
	astParser "github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/internal/run"
	"github.com/ergochat/readline"
	"github.com/sanity-io/litter"
)

var (
	defaultPrompt    = ansi.Magenta("> ") // ansi.Magenta("(repl)") + " > "
	incompletePrompt = ansi.Green("... ")
)

type Session struct {
	tokens      []lexer.Token // Incomplete/multiline tokens
	buf         []byte        // Multiline tokens
	interrupted bool          // If interrupted once
	done        bool
	multiline   bool            // Multiline editing enabled
	line        uint32          // Current line, greater than 0 if multiline
	evaluated   [][]lexer.Token // Successfully evaluated lines
	lastSaveLoc string          // Last provided path to 'save' command
	*readline.Instance
}

func NewSession() (*Session, error) {
	hist, err := HistoryFile()
	if err != nil {
		return nil, err
	}
	s := &Session{}
	rl, err := readline.NewFromConfig(&readline.Config{
		Prompt:              defaultPrompt,
		HistoryFile:         hist,
		InterruptPrompt:     ansi.Red("Ctrl+C"),
		EOFPrompt:           ansi.Red("exit"),
		FuncFilterInputRune: s.handleShortcut,
	})
	if err != nil {
		return nil, err
	}
	s.Instance = rl
	log.SetOutput(rl.Stderr())
	return s, nil
}

func Run(*command.Runner) {
	fmt.Println(ansi.Bold("Welcome to Klar"), ansi.Gray("v"+cli.KlarVersionAndCommit))
	ansi.ColorPrintfln(
		ansi.CodeGray,
		"Type %s for more information. Press %s or type %s to exit.",
		ansi.Cyan("help"), ansi.Cyan("Ctrl+D"), ansi.Cyan("exit"),
	)
	s, err := NewSession()
	if err != nil {
		cli.InternalError(err)
	}
	defer s.Close()
	for !s.done {
		s.Prompt()
	}
}

func HistoryFile() (string, error) {
	if err := module.LoadSystemDirs(); err != nil {
		return "", err
	}
	hist := filepath.Join(module.SystemDirs.Cache, "replHistory.txt")
	// Create the cache directory if missing
	if err := os.MkdirAll(module.SystemDirs.Cache, 0o755); err != nil {
		return hist, err
	}
	return hist, nil
}

func (s *Session) Prompt() {
	if s.multiline {
		s.line++
		s.SetPrompt(linePrompt(s.line))
	}
	input, err := s.ReadLine()
	switch err {
	case nil:
	case readline.ErrInterrupt:
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
		cli.Error("Failed to read input:", err)
	}
	s.interrupted = false
	if s.multiline {
		s.buf = append(s.buf, input...)
		s.buf = append(s.buf, '\n') // For continued lines
		s.checkMultilineEnd()
		return
	}
	tokens := tokenize(strings.NewReader(input), int64(len(input)/10))
	if len(tokens) > 1 && tokens[0].Kind == lexer.Identifier {
		if valid := s.handleCommand(tokens[0].Source, tokens[1:len(tokens)-1]); valid {
			return
		}
	}
	s.appendTokens(tokens)
	s.send()
}

func (s *Session) Error(msg string, err error) {
	ansi.TagFprintf(s.Stderr(), "<r! **>Error</><**>: %s:</**> %v\n", msg, err)
}

func (s *Session) send() {
	t := s.tokens
	if isIncomplete(t) {
		s.SetPrompt(incompletePrompt)
		return
	}
	s.SetPrompt(defaultPrompt)
	s.tokens = nil
	s.runTokens(t)
}

func (s *Session) runTokens(t []lexer.Token) {
	res, err := run.RunTokens(t, "repl")
	// TODO: get access to typechecked module in order to add Repl flag
	if err != nil {
		s.Error("Failed to evaluate", err)
		return
	}
	for _, mod := range res.Modules {
		if mod.Name() == "repl" {
			litter.Dump(mod.Programs["repl"])
		}
	}
	s.evaluated = append(s.evaluated, t)
}

func (s *Session) handleShortcut(r rune) (rune, bool) {
	const letterOffset = 'a' - 1
	switch r {
	case 'g' - letterOffset: // Ctrl+G
		s.handleCommand("multiline", nil)
		s.Prompt()
		return 0, false
	case 's' - letterOffset: // Ctrl+S
		s.handleCommand("save", nil)
		return 0, false
	case 'd' - letterOffset: // Ctrl+D
		s.handleCommand("exit", nil)
		return 0, false
	}
	return r, true
}

func (s *Session) handleCommand(cmd string, args []lexer.Token) (valid bool) {
	switch cmd {
	case "exit":
		s.Exit()
		return true
	case "help":
		s.PrintHelp()
	case "load":
		s.LoadFile(args)
	case "save":
		s.SaveFile(args)
	case "multiline", "ml":
		if s.multiline = !s.multiline; s.multiline {
			fmt.Fprintln(s.Stdout(), multilineEnabledMsg)
		} else {
			s.line = 0
			fmt.Fprintln(s.Stdout(), multilineDisabledMsg)
			s.sendMultiline()
		}
	default:
		return false
	}
	return true
}

func (s *Session) Exit() {
	s.Finish()
	s.Close()
	cli.Exit(0)
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
	trimmed := bytes.TrimSpace(s.buf)
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '.' {
		s.buf = s.buf[:len(s.buf)-2] // Remove newline and dot
		s.sendMultiline()
	}
}

func (s *Session) sendMultiline() {
	s.tokens = tokenize(bytes.NewReader(s.buf), int64(len(s.buf)/10))
	s.send()
	s.line = 0
	s.buf = nil
}

func tokenize(r io.Reader, cap int64) []lexer.Token {
	return lexer.NewLexer(r).TokenizeAll(cap)
}

func linePrompt(n uint32) string {
	return ansi.Magenta(fmt.Sprintf("%2d │ ", n))
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
