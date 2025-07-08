// Package klarml implements encoding and decoding of Klar Markup document structures (.klarml).
package klarml

import (
	"io"
	"slices"
)

// TokenizeReader reads a markup document from reader and returns tokens to be parsed.
func TokenizeReader(reader io.Reader) ([]Token, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return Tokenize(bytes), nil
}

// Tokenize reads from bytes and returns tokens to be parsed.
func Tokenize(bytes []byte) []Token {
	tokens := make([]Token, 0, len(bytes)/2)
	l := lexer{
		Bytes:    bytes,
		Index:    0,
		Position: Position{1, 1},
	}
	for l.HasBytes() {
		tokens = append(tokens, l.Tokenize())
	}
	tokens = append(tokens, newToken(l.Position, EOF, ""))
	tokens = slices.Clip(tokens)
	return tokens
}

// Parse reads a markup document from reader, returning the parsed abstract
// syntax tree (AST) and any errors that occured while parsing.
func Parse(bytes []byte) (Document, []error) {
	tokens := Tokenize(bytes)
	return ParseTokens(tokens)
}

// Parse reads a markup document from reader, returning the parsed abstract
// syntax tree (AST) and any errors that occured while parsing.
func ParseReader(reader io.Reader) (Document, []error) {
	tokens, err := TokenizeReader(reader)
	if err != nil {
		return Document{}, []error{err}
	}
	return ParseTokens(tokens)
}

// Parse converts tokens returned from [Tokenize] into an abstract
// syntax tree (AST), returning the parsed document and any errors that occured
// while parsing.
func ParseTokens(tokens []Token) (d Document, errors []error) {
	parserTokens := make([]Token, len(tokens))
	copy(parserTokens, tokens)
	p := parser{
		Index:  0,
		Tokens: parserTokens,
	}
	d.Comments = p.RemoveComments()
	return p.Parse()
}
