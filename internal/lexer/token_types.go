package lexer

type TokenType int

const (
	EOF            = iota
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

	Identifier
	Numeric
	Boolean
	Nil
	String
	Discard // _

	// Binary
	Plus     // +
	Minus    // -
	Times    // *
	Divide   // /
	Modulo   // %
	Exponent // ^

	// Assignment
	EqualSign  // =
	ColonEqual // :=
	PlusEqual  // +=
	MinusEqual // -=
	Increment  // ++
	Decrement  // --

	// Comparison
	Equals         // ==
	NotEqual       // !=
	GreaterThan    // >
	LessThan       // <
	GreaterEqualTo // >=
	LessEqualTo    // <=
	LogicalAnd     // &&
	LogicalOr      // ||
	LogicalNot     // !

	// Types
	Alternative // |
	Optional    // ?

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
	"++":  Increment,
	"--":  Decrement,
	"...": Spread,
	":=":  ColonEqual,
	"+=":  PlusEqual,
	"-=":  MinusEqual,
	"==":  Equals,
	"!=":  NotEqual,
	">=":  GreaterEqualTo,
	"<=":  LessEqualTo,
	"||":  LogicalOr,
	"&&":  LogicalAnd,
	"->":  Arrow,
	"=":   EqualSign,
	"+":   Plus,
	"-":   Minus,
	"*":   Times,
	"/":   Divide,
	"%":   Modulo,
	"^":   Exponent,
	"!":   LogicalNot,
	">":   GreaterThan,
	"<":   LessThan,
	"|":   Alternative,
	"?":   Optional,
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
	"_":      Discard,
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
