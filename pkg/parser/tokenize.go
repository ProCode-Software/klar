package parser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// TokenizeFile reads from a file and converts it into lexer tokens.
func TokenizeFile(file *os.File, includeComments bool) ([]lexer.Token, error) {
	// Estimate token capacity
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	byteSize := stat.Size()
	return Tokenize(file, includeComments, byteSize/4)
}

// Tokenize reads from reader and converts it into lexer tokens.
func Tokenize(
	reader io.Reader, includeComments bool, sizeEstimate int64,
) (tokens []lexer.Token, err error) {
	lex := lexer.NewLexer(reader)
	lex.IncludeComments = includeComments
	tokens = make([]lexer.Token, 0, sizeEstimate)

	// Recover if panics
	defer func() {
		if err2 := recover(); err2 != nil {
			err = fmt.Errorf("%v", err2)
		}
	}()
	for {
		token := lex.Tokenize()
		tokens = append(tokens, *token)
		if token.Kind == lexer.EOF {
			break
		}
	}
	return tokens, nil
}

// TokenizeString reads from a string and converts it into lexer tokens.
func TokenizeString(source string, includeComments bool) ([]lexer.Token, error) {
	file := strings.NewReader(source)
	return Tokenize(file, includeComments, int64(len(source)/3))
}
