package jsir

type Expression interface{ _expr() }

type BinaryExpression struct {
	Left, Right Expression
	Operator    Operator
}

type UnaryExpression struct {
	Operator Operator
	Operand  Expression
}

type (
	Symbol           struct{ Name string }
	NumericLiteral   struct{ Value float64 }
	BooleanLiteral   struct{ Value bool }
	NullLiteral      struct{}
	UndefinedLiteral struct{}
)

type StringLiteral struct {
	QuoteStyle QuoteStyle
	Content    string
}

type QuoteStyle uint8

const (
	SingleQuote QuoteStyle = iota
	DoubleQuote
)

type TemplateLiteral struct {
	Tag Expression // Can be nil
	// TODO: How to store fragments
}

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
	Entries [2]Expression
}
