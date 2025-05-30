package lexer

import "io"

//go:generate stringer -type=TokenType
type TokenType int

const (
	_ TokenType = iota
	EOF
	EndOfStatement // Replacement for semicolons
	Illegal
	Newline

	// Punctuation
	Comma              // ,
	Dot                // .
	LineComment        // //
	BlockComment       // /*
	Colon              // :
	LeftBracket        // [
	RightBracket       // ]
	LeftParenthesis    // (
	RightParenthesis   // )
	LeftCurlyBrace     // {
	RightCurlyBrace    // }
	HashLeftCurlyBrace // #{
	At                 // @
	Hash               // #

	Identifier
	Numeric
	Boolean
	Nil
	String
	Underscore // _

	// Binary
	Plus     // +
	Minus    // -
	Asterisk // *
	Slash    // /
	Percent  // %
	Caret    // ^

	// Assignment
	Equal      // =
	ColonEqual // :=
	PlusEqual  // +=
	MinusEqual // -=
	PlusPlus   // ++
	MinusMinus // --

	// Comparison
	EqualEqual     // ==
	NotEqual       // !=
	GreaterThan    // >
	LessThan       // <
	GreaterEqualTo // >=
	LessEqualTo    // <=
	AndAnd         // &&
	OrOr           // ||
	Not            // !

	// Types
	Stroke   // |
	Question // ?

	// Misc
	Ellipsis // ...
	Arrow    // ->
	Pipeline // |>

	// Keywords
	And
	For
	Func
	Import
	In
	Next
	Or
	Public
	Return
	Type
	When
)

const (
	NumberFormatDecimal = iota
	NumberFormatHex
	NumberFormatOctal
	NumberFormatBinary
)

var OperatorMap = map[string]TokenType{
	"++":  PlusPlus,
	"--":  MinusMinus,
	"...": Ellipsis,
	":=":  ColonEqual,
	"+=":  PlusEqual,
	"-=":  MinusEqual,
	"==":  EqualEqual,
	"!=":  NotEqual,
	">=":  GreaterEqualTo,
	"<=":  LessEqualTo,
	"||":  OrOr,
	"&&":  AndAnd,
	"->":  Arrow,
	"=":   Equal,
	"+":   Plus,
	"-":   Minus,
	"*":   Asterisk,
	"/":   Slash,
	"%":   Percent,
	"^":   Caret,
	"!":   Not,
	">":   GreaterThan,
	"<":   LessThan,
	"|":   Stroke,
	"?":   Question,
	"|>":  Pipeline,

	// Punctuation
	":":  Colon,
	".":  Dot,
	"#":  Hash,
	"@":  At,
	"#{": HashLeftCurlyBrace,
	"//": LineComment,
	"/*": BlockComment,
}

var KeywordMap = map[string]TokenType{
	"and":    And,
	"for":    For,
	"func":   Func,
	"import": Import,
	"in":     In,
	"next":   Next,
	"or":     Or,
	"public": Public,
	"return": Return,
	"type":   Type,
	"when":   When,
	"_":      Underscore,
	"true":   Boolean,
	"false":  Boolean,
	"nil":    Nil,
}

func NewToken(pos Position, kind TokenType, src string) *Token {
	return &Token{pos, kind, src, nil}
}

type Token struct {
	Position
	Kind       TokenType
	Source     string
	Attributes map[string]any
}

func (t *Token) SetAttribute(key string, value any) *Token {
	if t.Attributes == nil {
		t.Attributes = make(map[string]any)
	}
	t.Attributes[key] = value
	return t
}

func (t TokenType) LitterDump(w io.Writer) {
	w.Write([]byte("{" + t.String() + "}"))
}
