package lexer

//go:generate stringer -type=TokenType
type TokenType int

const (
	EOF            TokenType = iota
	EndOfStatement           // Replacement for semicolons
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
	Spread   // ...
	Arrow    // ->
	Pipeline // |>

	// Keywords
	For
	Func
	Import
	Next
	Return
	Type
	When
)

const (
	NumberFormatDecimal = iota
	NumberFormatHexadecimal
	NumberFormatOctal
	NumberFormatBinary
)

var OperatorMap = map[string]TokenType{
	"++":  PlusPlus,
	"--":  MinusMinus,
	"...": Spread,
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
	"@":  At,
	"#{": HashLeftCurlyBrace,
	"//": LineComment,
	"/*": BlockComment,
}

var KeywordMap = map[string]TokenType{
	"for":    For,
	"func":   Func,
	"import": Import,
	"next":   Next,
	"return": Return,
	"type":   Type,
	"when":   When,
	"_":      Underscore,
	"true":   Boolean,
	"false":  Boolean,
	"nil":    Nil,
}

func NewLexerToken(pos Position, kind TokenType, src string) *Token {
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
