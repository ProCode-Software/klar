package parser

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

// BindingPower represents the operator precedence for a type of operator.
type BindingPower int

// TypeScript Reference:
// 	https://github.com/microsoft/typescript-go/blob/main/internal/ast/precedence.go

const (
	DefaultBindingPower        BindingPower = iota
	CommaBindingPower                       // ,
	AssignBindingPower                      // :=, +=, -=, =
	ExpressionBindingPower                  // Minimum for expressions
	LambdaBindingPower                      // ->
	ObjectPipelineBindingPower              // |.
	LogicalBindingPower                     // ||, &&
	PipelineBindingPower                    // |>

	RelationalBindingPower     // ==, !=, >, <, <=, >=, in, !in
	DistributiveBindingPower   // and, or
	RangeBindingPower          // ..., ..<
	AdditiveBindingPower       // +, -
	MultiplicativeBindingPower // *, /, %
	UnaryBindingPower          // left ..., !, ++, -- (prefix operators aren't LEDs)
	ExponentiationBindingPower // ^ (higher than unary: -2 ^ 3 = -(2 ^ 3))
	CallBindingPower           // Call: (
	MemberBindingPower         // Index/Slice: . [
	PrimaryBindingPower        // Primary expressions, such as literals
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

	lexer.Pipeline: PipelineBindingPower,

	lexer.StrokeDot: ObjectPipelineBindingPower,

	lexer.LessThan:       RelationalBindingPower,
	lexer.GreaterThan:    RelationalBindingPower,
	lexer.LessEqualTo:    RelationalBindingPower,
	lexer.GreaterEqualTo: RelationalBindingPower,
	lexer.EqualEqual:     RelationalBindingPower,
	lexer.NotEqual:       RelationalBindingPower,
	lexer.In:             RelationalBindingPower,
	lexer.NotIn:          RelationalBindingPower,

	lexer.Ellipsis:       RangeBindingPower,
	lexer.DotDotLessThan: RangeBindingPower,

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

	// NUDs aren't important for precedence, but when used as a LED,
	// you can get a better unexpected token error.
	lexer.String:     PrimaryBindingPower,
	lexer.Numeric:    PrimaryBindingPower,
	lexer.Boolean:    PrimaryBindingPower,
	lexer.Identifier: PrimaryBindingPower,
	lexer.Nil:        PrimaryBindingPower,
	lexer.Underscore: PrimaryBindingPower,
	lexer.Regex:      PrimaryBindingPower,
}

const (
	_ BindingPower = AssignBindingPower + iota
	DefaultTypeBindingPower
	FunctionTypeBindingPower  // ->
	VariadicTypeBindingPower  // ...
	OptionalTypeBindingPower  // ?
	UnionTypeBindingPower     // |
	NamespaceTypeBindingPower // .
	GenericTypeBindingPower   // <
	PrimaryTypeBindingPower   // Types
)

var TypeBindingPowerMap = map[lexer.TokenType]BindingPower{
	lexer.Arrow:    FunctionTypeBindingPower,
	lexer.Question: OptionalTypeBindingPower,
	lexer.Stroke:   UnionTypeBindingPower,
	lexer.LessThan: GenericTypeBindingPower,
	lexer.Dot:      NamespaceTypeBindingPower,

	lexer.Identifier: PrimaryTypeBindingPower,
	lexer.Underscore: PrimaryTypeBindingPower,
}
