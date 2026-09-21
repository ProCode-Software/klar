package jsir

type Expression interface{ _expr() }

// Many of these are in
// https://tc39.es/ecma262/multipage/ecmascript-language-expressions.html

type AssignmentExpression struct {
	Assignee Expression // [Destructure] or index
	Operator Operator
	Value    Expression
}

type BinaryExpression struct {
	Left, Right Expression
	Operator    Operator
}

type UnaryExpression struct {
	Operand  Expression
	Operator Operator
}

type PostfixUnaryExpression UnaryExpression

type (
	Symbol           struct{ Name string }
	NumericLiteral   struct{ Value float64 }
	BooleanLiteral   struct{ Value bool }
	NullLiteral      struct{}
	UndefinedLiteral struct{}
)

type StringLiteral struct {
	QuoteStyle QuoteStyle
	Content    string // Characters will be escaped as needed
}

type QuoteStyle uint8

const (
	AutoQuote   QuoteStyle = iota // Quote to escape the least
	SingleQuote            = '\''
	DoubleQuote            = '"'
)

// https://tc39.es/ecma262/multipage/ecmascript-language-expressions.html#sec-template-literals
type TemplateLiteral struct {
	Tag Expression // Can be nil
	// TODO: How to store fragments
}

// Allowed in [ArrayLiteral], [CallExpression], and [ObjectLiteral] as an entry key.
type SpreadExpression struct {
	Right Expression
}

type ArrayLiteral struct {
	Items []Expression
}

type ObjectLiteral struct {
	// The first represents the key. Will be put in brackets unless it is a
	// string or numeric literal. The second expression may be nil if
	// it uses a shorthand.
	//
	// If the entry is a spread, the key will be a [SpreadExpression] and
	// the value will be nil.
	Entries [][2]Expression
}

type RegExpLiteral struct {
	Pattern, Flags string
}

type IndexExpression = MemberExpression

type MemberExpression struct {
	Object   Expression
	Property Expression // A [Symbol] if !Computed
	Computed bool       // If brackets were used
	// If '?.' or '?.[' was used
	// See https://tc39.es/ecma262/multipage/ecmascript-language-expressions.html#prod-OptionalChain
	Optional bool
}

type CallExpression struct {
	Callee    Expression
	Arguments []Expression
}

type TernaryExpression struct {
	Condition, True, False Expression
}

// TODO: This expression also has an operator precedence
type CommaExpression struct {
	Items []Expression // At least 2
}

type ArrowFunction struct {
	Async  bool
	Params []Expression
	Body   Expression
}
