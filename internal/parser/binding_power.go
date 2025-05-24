package parser

import "github.com/ProCode-Software/klar/internal/lexer"

// BindingPower represents the operator precedence for a type of operator.
type BindingPower int

// References:
// - https://github.com/microsoft/typescript-go/blob/main/internal/ast/precedence.go
// - https://github.com/microsoft/typescript-go/blob/main/internal/parser/parser.go
const (
	DefaultBindingPower        BindingPower = iota
	CommaBindingPower                       // ,
	AssignBindingPower                      // :=, +=, -=, =
	LogicalBindingPower                     // ||, &&, | or + in type
	RelationalBindingPower                  // ==, >, etc.
	AdditiveBindingPower                    // + and -
	MultiplicativeBindingPower              // *, /, %
	ExponentialBindingPower                 // ^
	UnaryBindingPower                       // Prefix/Suffix: + -
	CallBindingPower                        // Call: (
	MemberBindingPower                      // Index: . [
	PrimaryBindingPower                     // Primary expressions, such as literals
)

var BindingPowerMap = map[lexer.TokenType]BindingPower{
	lexer.Comma: CommaBindingPower,

	lexer.Colon:      AssignBindingPower,
	lexer.ColonEqual: AssignBindingPower,
	lexer.Equal:      AssignBindingPower,
	lexer.PlusEqual:  AssignBindingPower,
	lexer.MinusEqual: AssignBindingPower,

	lexer.LessThan:       RelationalBindingPower,
	lexer.GreaterThan:    RelationalBindingPower,
	lexer.LessEqualTo:    RelationalBindingPower,
	lexer.GreaterEqualTo: RelationalBindingPower,
	lexer.EqualEqual:     RelationalBindingPower,
	lexer.NotEqual:       RelationalBindingPower,
	lexer.Spread:         RelationalBindingPower, // Infix only: 1...10

	lexer.AndAnd: LogicalBindingPower,
	lexer.OrOr:   LogicalBindingPower,
	lexer.Stroke: LogicalBindingPower, // In when statements, a bit lower than logical, but higher than comma

	lexer.Plus:  AdditiveBindingPower,
	lexer.Minus: AdditiveBindingPower,

	lexer.Asterisk: MultiplicativeBindingPower,
	lexer.Slash:    MultiplicativeBindingPower,
	lexer.Percent:  MultiplicativeBindingPower,

	lexer.Caret: ExponentialBindingPower,

	lexer.LeftParenthesis: CallBindingPower,

	lexer.Dot:         MemberBindingPower,
	lexer.LeftBracket: MemberBindingPower,

	lexer.String:     PrimaryBindingPower,
	lexer.Numeric:    PrimaryBindingPower,
	lexer.Boolean:    PrimaryBindingPower,
	lexer.Identifier: PrimaryBindingPower,
}

const (
	_ BindingPower = iota
	DefaultTypeBindingPower
	FunctionTypeBindingPower
	UnionTypeBindingPower
	OptionalTypeBindingPower
	PrimaryTypeBindingPower
)

var TypeBindingPowerMap = map[lexer.TokenType]BindingPower{
	lexer.Arrow:      FunctionTypeBindingPower,
	lexer.Stroke:     UnionTypeBindingPower,
	lexer.Plus:       UnionTypeBindingPower,
	lexer.Question:   OptionalTypeBindingPower,
	lexer.String:     PrimaryTypeBindingPower,
	lexer.Numeric:    PrimaryTypeBindingPower,
	lexer.Boolean:    PrimaryTypeBindingPower,
	lexer.Identifier: PrimaryTypeBindingPower,
	lexer.Spread:     PrimaryTypeBindingPower,
}
