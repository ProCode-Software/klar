// New keywords and operators are defined in this file.

package lexer

// TokenType represent a lexer token
type TokenType int

const (
	_ TokenType = iota
	EOF
	EndOfStatement // Replacement for semicolons
	Illegal        // Unknown character
	Newline        // Source only -- replaced with EndOfStatement

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
	Regex      // @/
	Underscore // _

	// Arithmetic
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
	NotIn          // !in
	NotCan         // !can

	// Types
	Stroke   // |
	Question // ?

	// Misc
	Ellipsis       // ...
	DotDotLessThan // ..<
	Arrow          // ->
	Pipeline       // |>
	StrokeDot      // |.
	Backslash      // \

	// Keywords
	And
	Await
	Can
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

var OperatorMap = map[string]TokenType{
	"...": Ellipsis,
	"..<": DotDotLessThan,
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
	"|.":  StrokeDot,
	`\`:   Backslash,

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
}

var KeywordMap = map[string]TokenType{
	"and":    And,
	"await":  Await,
	"stop":   Stop,
	"can":    Can,
	"for":    For,
	"func":   Func,
	"go":     Go,
	"import": Import,
	"if":     If,
	"in":     In,
	"next":   Next,
	"opaque": Opaque,
	"or":     Or,
	"public": Public,
	"return": Return,
	"try":    Try,
	"type":   Type,
	"when":   When,
	"while":  While,

	"_":     Underscore,
	"true":  Boolean,
	"false": Boolean,
	"nil":   Nil,
}
