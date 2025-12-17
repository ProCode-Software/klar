// Package klon implements encoding and decoding of Klar Markup document structures (.klon).
package klon

import (
	"io"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/parser"
)

// TokenizeReader reads a markup document from reader and returns tokens to be parsed
// and any error that occured while reading.
func TokenizeReader(reader io.Reader) ([]parser.Token, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return Tokenize(bytes), nil
}

// Tokenize reads from bytes and returns tokens to be parsed.
func Tokenize(bytes []byte) []parser.Token {
	return parser.Tokenize(bytes)
}

// Parse reads a markup document from reader, returning the parsed abstract
// syntax tree (AST) and any errors that occured while parsing.
func Parse(bytes []byte) (*ast.Document, []error) {
	tokens := Tokenize(bytes)
	return parser.ParseTokens(tokens)
}

// Parse reads a markup document from reader, returning the parsed abstract
// syntax tree (AST) and any errors that occured while reading or parsing.
func ParseReader(reader io.Reader) (*ast.Document, []error) {
	tokens, err := TokenizeReader(reader)
	if err != nil {
		return nil, []error{err}
	}
	return parser.ParseTokens(tokens)
}

// Parse converts tokens returned from [Tokenize] into an abstract
// syntax tree (AST), returning the parsed document and any errors that occured
// while parsing.
func ParseTokens(tokens []parser.Token) (d *ast.Document, errors []error) {
	return parser.ParseTokens(tokens)
}
