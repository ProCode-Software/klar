package parser

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

// BindingPower represents the operator precedence for a type of operator.
type BindingPower int

// Reference: https://github.com/microsoft/typescript-go/blob/main/internal/ast/precedence.go
const (
	DefaultBindingPower        BindingPower = iota
	CommaBindingPower                       // ,
	AssignBindingPower                      // :=, +=, -=, =
	ExpressionBindingPower                  // Minimum for expressions
	LambdaBindingPower                      // ->
	LogicalBindingPower                     // ||, &&, | or + in type
	RelationalBindingPower                  // ==, >, etc.
	DistributiveBindingPower                // and, or
	RangeBindingPower                       // ...
	AdditiveBindingPower                    // + and -
	MultiplicativeBindingPower              // *, /, %
	UnaryBindingPower                       // Prefix/Suffix: + -
	ExponentiationBindingPower              // ^
	CallBindingPower                        // Call: (
	MemberBindingPower                      // Index/Slice: . [
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

	lexer.AndAnd:   LogicalBindingPower,
	lexer.OrOr:     LogicalBindingPower,
	lexer.Pipeline: LogicalBindingPower,

	lexer.LessThan:       RelationalBindingPower,
	lexer.GreaterThan:    RelationalBindingPower,
	lexer.LessEqualTo:    RelationalBindingPower,
	lexer.GreaterEqualTo: RelationalBindingPower,
	lexer.EqualEqual:     RelationalBindingPower,
	lexer.NotEqual:       RelationalBindingPower,
	lexer.In:             RelationalBindingPower,

	lexer.Ellipsis: RangeBindingPower,

	lexer.And: DistributiveBindingPower,
	lexer.Or:  DistributiveBindingPower,

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
	lexer.Underscore: PrimaryBindingPower,
}

const (
	_ BindingPower = AssignBindingPower + iota
	DefaultTypeBindingPower
	FunctionTypeBindingPower
	VariadicTypeBindingPower
	OptionalTypeBindingPower
	UnionTypeBindingPower
	GenericTypeBindingPower
	MemberTypeBindingPower
	PrimaryTypeBindingPower
)

var TypeBindingPowerMap = map[lexer.TokenType]BindingPower{
	lexer.Arrow:    FunctionTypeBindingPower,
	lexer.Ellipsis: VariadicTypeBindingPower,
	lexer.Question: OptionalTypeBindingPower,
	lexer.Stroke:   UnionTypeBindingPower,
	lexer.LessThan: GenericTypeBindingPower,
	lexer.Dot:      MemberTypeBindingPower,

	lexer.Boolean:    PrimaryTypeBindingPower,
	lexer.Identifier: PrimaryTypeBindingPower,
	lexer.Underscore: PrimaryTypeBindingPower,
}
