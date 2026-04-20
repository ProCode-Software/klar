// New keywords and operators are defined in this file.

package lexer

// TokenType represent a lexer token.
type TokenType int

const (
	_ TokenType = iota
	EOF
	Newline // Newlines during lexing; "semicolons" during parsing
	Illegal // Unknown character

	// Punctuation
	Comma              // ,
	Dot                // .
	LineComment        // //
	BlockComment       // /*
	Hashbang           // #!
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
	Regex      // #/
	Underscore // _

	// Arithmetic
	Plus     // +
	Minus    // -
	Asterisk // *
	Slash    // /
	Percent  // %
	Caret    // ^

	// Assignment
	Equal         // =
	ColonEqual    // :=
	PlusEqual     // +=
	MinusEqual    // -=
	AsteriskEqual // *=
	SlashEqual    // /=
	PercentEqual  // %=
	CaretEqual    // ^=

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
	NotNot         // !!
	NotIn          // !in

	// Types
	Stroke   // |
	Question // ?

	// Misc
	Ellipsis       // ...
	DotDotLessThan // ..<
	Arrow          // ->
	Pipeline       // |>
	StrokeDot      // |.

	// Keywords
	And
	As
	Await
	For
	Func
	Go
	If
	Import
	In
	Next
	Opaque
	Or
	Public
	Return
	Stop
	Try
	Type
	When
	While
)

// List of operators in the Klar language. Punctuation is also included in this map.
var OperatorMap = map[string]TokenType{
	"...": Ellipsis,
	"..<": DotDotLessThan,
	":=":  ColonEqual,
	"+=":  PlusEqual,
	"-=":  MinusEqual,
	"*=":  AsteriskEqual,
	"/=":  SlashEqual,
	"%=":  PercentEqual,
	"^=":  CaretEqual,
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
	"!!":  NotNot,
	">":   GreaterThan,
	"<":   LessThan,
	"|":   Stroke,
	"?":   Question,
	"|>":  Pipeline,
	"|.":  StrokeDot,

	// Punctuation
	":":  Colon,
	".":  Dot,
	"#":  Hash,
	"@":  At,
	",":  Comma,
	"[":  LeftBracket,
	"]":  RightBracket,
	"(":  LeftParenthesis,
	")":  RightParenthesis,
	"{":  LeftCurlyBrace,
	"}":  RightCurlyBrace,
	"#{": HashLeftCurlyBrace,
	"//": LineComment,
	"/*": BlockComment,
	"#!": Hashbang,
	"#/": Regex,
}

// List of keywords in the Klar language. All keys are reserved and cannot be used
// as identifiers.
var KeywordMap = map[string]TokenType{
	"and":    And,
	"as":     As,
	"await":  Await,
	"for":    For,
	"func":   Func,
	"go":     Go,
	"import": Import,
	"if":     If,
	"in":     In,
	"!in":    NotIn,
	"next":   Next,
	"opaque": Opaque,
	"or":     Or,
	"public": Public,
	"return": Return,
	"stop":   Stop,
	"try":    Try,
	"type":   Type,
	"when":   When,
	"while":  While,

	"_":     Underscore,
	"true":  Boolean,
	"false": Boolean,
	"nil":   Nil,
}
