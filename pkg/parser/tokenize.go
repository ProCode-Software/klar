package parser

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

type LexerFlags = lexer.Flags

const (
	// Include comments when tokenizing. Useful for documentation parsing.
	IncludeComments = lexer.IncludeComments
)

// TokenizeFile reads from file and converts it into lexer tokens.
func TokenizeFile(file *os.File, flags LexerFlags) ([]lexer.Token, error) {
	// Estimate token capacity
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	byteSize := stat.Size()
	return Tokenize(file, flags, byteSize/10)
}

func TokenizeLexer(l *lexer.Lexer, cap int64) (tokens []lexer.Token, err error) {
	if cap > 0 {
		tokens = make([]lexer.Token, 0, cap)
	} else {
		tokens = make([]lexer.Token, 0)
	}
	// Recover if the lexer panics (read error)
	defer func() {
		if r := recover(); r != nil {
			err, _ = r.(error)
		}
	}()
	for {
		token := l.Tokenize()
		tokens = append(tokens, *token)
		if token.Kind == lexer.EOF {
			break
		}
	}
	return tokens, nil
}

// Tokenize reads from r and converts it into lexer tokens.
func Tokenize(r io.Reader, flags LexerFlags, cap int64) (
	tokens []lexer.Token, err error,
) {
	lex := lexer.NewLexer(r, flags)
	return TokenizeLexer(lex, cap)
}

// TokenizeString reads from src and converts it into lexer tokens.
func TokenizeString(src string, flags LexerFlags) ([]lexer.Token, error) {
	file := strings.NewReader(src)
	return Tokenize(file, flags, int64(len(src)/3))
}

// TokenizeBytes reads from b and converts it into lexer tokens.
func TokenizeBytes(b []byte, flags LexerFlags) ([]lexer.Token, error) {
	file := bytes.NewReader(b)
	return Tokenize(file, flags, int64(len(b)/3))
}

// AddSemicolons returns tokens with all [lexer.Newline] tokens either replaced with
// [lexer.EndOfStatement] or removed. All comments are also removed from tokens.
func AddSemicolons(tokens []lexer.Token) []lexer.Token {
	p := parser.New(tokens, nil)
	p.InsertEOS()
	return p.Tokens
}
