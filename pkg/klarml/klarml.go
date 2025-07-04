package klarml

import "io"

func Tokenize(reader io.Reader) []Token {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
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
	return tokens
}
