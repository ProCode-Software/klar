package lexer

type TokenType int

const (
	EOF = iota
	Illegal
	Newline

	// Punctuation
	Comma            // ,
	Dot              // .
	Colon            // :
	LeftBracket      // [
	RightBracket     // ]
	LeftParenthesis  // (
	RightParenthesis // )
	LeftCurlyBrace   // {
	RightCurlyBrace  // }

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
	TypeOption  // ?

	// Misc
	Spread // ...
	Arrow  // ->

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
	Decimal = iota
	Hexadecimal
	Octal
	Binary
)

var Operators = []string{
	"++", "--", "...",
	":=", "+=", "-=",
	"==", "!=", ">=", "<=", "||", "&&",
	"=", "+", "-", "*", "/", "%", "^", "!", ">", "<",
	"|", "?", "->",
}

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
	"?":   TypeOption,
	":":   Colon, // Punctuation
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
	"true": Boolean,
	"false": Boolean,
	"nil": Nil,
}
