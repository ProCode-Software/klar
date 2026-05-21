package klon

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type TokenType uint8

const (
	Illegal TokenType = iota
	EOF
	Newline
	Identifier
	Number
	String
	Boolean
	None
	At // @
	Colon
	Comma
	Dash  // -
	Arrow // <-
	Comment
	Variable
	LeftBracket
	RightBracket
	LeftCurly
	RightCurly
)

type attrs = map[string]any

type Token struct {
	Kind   TokenType
	Src    string
	Pos    lexer.Position
	BufPos int // Position in reader buffer
	Attrs  map[string]any
}

func (tok Token) Range() ranges.Range {
	return ranges.Range{Start: tok.Pos, End: tok.End()}
}

func (tok *Token) End() lexer.Position {
	if tok.Attrs != nil {
		if end, ok := tok.Attrs["end"]; ok {
			return end.(lexer.Position)
		}
	}
	pos := tok.Pos
	pos.Col += uint32(len(tok.Src))
	return pos
}
