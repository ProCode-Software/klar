// New keywords and operators are defined in this file.

package lexer

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
	Ellipsis       // ...
	DotDotLessThan // ..<
	Arrow          // ->
	Pipeline       // |>
	StrokeDot      // |.
	Backslash      // \

	// Keywords
	And
	Break
	Can
	For
	Func
	Go
	Import
	In
	Next
	NotCan // !can
	NotIn  // !in
	Opaque
	Or
	Public
	Return
	Type
	When
	While
)

var OperatorMap = map[string]TokenType{
	"++":   PlusPlus,
	"--":   MinusMinus,
	"...":  Ellipsis,
	"..<":  DotDotLessThan,
	":=":   ColonEqual,
	"+=":   PlusEqual,
	"-=":   MinusEqual,
	"==":   EqualEqual,
	"!=":   NotEqual,
	">=":   GreaterEqualTo,
	"<=":   LessEqualTo,
	"||":   OrOr,
	"&&":   AndAnd,
	"->":   Arrow,
	"=":    Equal,
	"+":    Plus,
	"-":    Minus,
	"*":    Asterisk,
	"/":    Slash,
	"%":    Percent,
	"^":    Caret,
	"!":    Not,
	"!in":  NotIn,
	"!can": NotCan,
	">":    GreaterThan,
	"<":    LessThan,
	"|":    Stroke,
	"?":    Question,
	"|>":   Pipeline,
	"|.":   StrokeDot,
	`\`:    Backslash,

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
	"break":  Break,
	"can":    Can,
	"for":    For,
	"func":   Func,
	"go":     Go,
	"import": Import,
	"in":     In,
	"next":   Next,
	"opaque": Opaque,
	"or":     Or,
	"public": Public,
	"return": Return,
	"type":   Type,
	"when":   When,
	"while":  While,

	"_":     Underscore,
	"true":  Boolean,
	"false": Boolean,
	"nil":   Nil,
}
