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
	ExpressionBindingPower                  // Minimum for expressions
	LambdaBindingPower                      // ->
	LogicalBindingPower                     // ||, &&, | or + in type
	RelationalBindingPower                  // ==, >, etc.
	AdditiveBindingPower                    // + and -
	MultiplicativeBindingPower              // *, /, %
	UnaryBindingPower                       // Prefix/Suffix: + -
	ExponentiationBindingPower              // ^
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

	lexer.Arrow: LambdaBindingPower,

	lexer.AndAnd: LogicalBindingPower,
	lexer.OrOr:   LogicalBindingPower,
	lexer.Stroke: LogicalBindingPower, // In when statements, a bit lower than logical, but higher than comma

	lexer.LessThan:       RelationalBindingPower,
	lexer.GreaterThan:    RelationalBindingPower,
	lexer.LessEqualTo:    RelationalBindingPower,
	lexer.GreaterEqualTo: RelationalBindingPower,
	lexer.EqualEqual:     RelationalBindingPower,
	lexer.NotEqual:       RelationalBindingPower,
	lexer.Spread:         RelationalBindingPower, // Infix only: 1...10

	lexer.Plus:  AdditiveBindingPower,
	lexer.Minus: AdditiveBindingPower,

	lexer.Asterisk: MultiplicativeBindingPower,
	lexer.Slash:    MultiplicativeBindingPower,
	lexer.Percent:  MultiplicativeBindingPower,

	lexer.PlusPlus:   UnaryBindingPower,
	lexer.MinusMinus: UnaryBindingPower,

	lexer.Caret: ExponentiationBindingPower,

	lexer.LeftParenthesis: CallBindingPower,

	lexer.Dot:         MemberBindingPower,
	lexer.LeftBracket: MemberBindingPower,

	lexer.String:     PrimaryBindingPower,
	lexer.Numeric:    PrimaryBindingPower,
	lexer.Boolean:    PrimaryBindingPower,
	lexer.Identifier: PrimaryBindingPower,
	lexer.Nil:        PrimaryBindingPower,
}

const (
	_ BindingPower = iota + AssignBindingPower
	DefaultTypeBindingPower
	FunctionTypeBindingPower
	UnionTypeBindingPower
	OptionalTypeBindingPower
	PrimaryTypeBindingPower
)

var TypeBindingPowerMap = map[lexer.TokenType]BindingPower{
	lexer.Arrow:       FunctionTypeBindingPower,
	lexer.Stroke:      UnionTypeBindingPower,
	lexer.Plus:        UnionTypeBindingPower,
	lexer.Question:    OptionalTypeBindingPower,
	lexer.String:      PrimaryTypeBindingPower,
	lexer.Numeric:     PrimaryTypeBindingPower,
	lexer.Boolean:     PrimaryTypeBindingPower,
	lexer.Identifier:  PrimaryTypeBindingPower,
	lexer.Spread:      PrimaryTypeBindingPower,
	lexer.LessEqualTo: PrimaryTypeBindingPower,
}
