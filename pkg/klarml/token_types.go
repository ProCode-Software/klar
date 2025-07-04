package klarml

type TokenType int

const (
	EOF TokenType = iota
	Newline

	Identifier // name
	Version    // v1.0, v2.0-beta-1
	Numeric    // 3, 3.0
	Boolean    // true, false
	Namespace  // @namespace
	String     // " or '
	Comment    // /* or //

	Colon      // :
	Hyphen     // -
	Dollar     // $
	LeftBrace  // {
	RightBrace // }
	Period     // .
	Equal      // =
)

type Token struct {
	Position   Position
	Kind       TokenType
	Source     string
	Attributes any
}
type Position struct {
	Line, Col int
}

type StringAttrs struct {
	Unterminated bool
	Unquoted     bool
	QuoteStyle   byte
}
type CommentAttrs struct {
	Block        bool
	Unterminated bool
}
