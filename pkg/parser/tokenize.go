package parser

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

// TokenizeFile reads from file and converts it into lexer tokens.
func TokenizeFile(file *os.File) []lexer.Token {
	// Estimate token capacity
	var est int64
	if stat, err := file.Stat(); err == nil {
		est = stat.Size() / 10
	}
	return Tokenize(file, est)
}

func TokenizeLexer(l *lexer.Lexer, cap int64) (tokens []lexer.Token) {
	if cap > 0 {
		tokens = make([]lexer.Token, 0, cap)
	} else {
		tokens = make([]lexer.Token, 0)
	}
	// Recover if the lexer panics (read error)
	for {
		token := l.Tokenize()
		tokens = append(tokens, *token)
		if token.Kind == lexer.EOF {
			break
		}
	}
	return tokens
}

// Tokenize reads from r and converts it into lexer tokens.
func Tokenize(r io.Reader, cap int64) []lexer.Token {
	lex := lexer.NewLexer(r)
	return TokenizeLexer(lex, cap)
}

// TokenizeString reads from src and converts it into lexer tokens.
func TokenizeString(src string) []lexer.Token {
	file := strings.NewReader(src)
	return Tokenize(file, int64(len(src)/3))
}

// TokenizeBytes reads from b and converts it into lexer tokens.
func TokenizeBytes(b []byte) []lexer.Token {
	file := bytes.NewReader(b)
	return Tokenize(file, int64(len(b)/3))
}

// InsertEOS returns tokens with [lexer.Newline] tokens where a newline terminates
// a statement. It removes comments and raw newline tokens that do not terminate
// a statement.
func InsertEOS(tokens []lexer.Token) []lexer.Token {
	p := parser.New(tokens, nil)
	p.InsertEOS()
	return p.Tokens
}
