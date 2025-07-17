package parser

import pkglex "github.com/ProCode-Software/klar/internal/lexer"

// The kind of token read by the tokenizer is one of these values.
type TokenType int

const (
	EOF TokenType = iota
	Newline

	Identifier     // name
	Numeric        // 3, 3.0
	TokenNamespace // @namespace
	String         // " or '
	TokenComment   // /* or //

	Colon      // :
	Comma      // ,
	Hyphen     // -
	Dollar     // $
	LeftBrace  // {
	RightBrace // }
	Period     // .
)

// A Token is the result of [lexer.Tokenize].
type Token struct {
	Position   Position
	Kind       TokenType
	Source     string
	Attributes any
}

// Position represents a line-column position of a token in a file.
type Position = pkglex.Position

type StringAttrs struct {
	Unterminated bool
	Unquoted     bool
	QuoteStyle   byte
}
type CommentAttrs struct {
	Block        bool
	Unterminated bool
}
