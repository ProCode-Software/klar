package parser

import "github.com/ProCode-Software/klar/internal/lexer"

// BindingPower represents the operator precedence for a type of operator.
type BindingPower int

const (
	DefaultBindingPower        BindingPower = iota
	CommaBindingPower                       // ,
	AssignBindingPower                      // :=, +=, -=, =
	LogicalBindingPower                     // || and &&
	RelationalBindingPower                  // ==, >, etc.
	AdditiveBindingPower                    // + and -
	MultiplicativeBindingPower              // * and /
	UnaryBindingPower
	CallBindingPower
	MemberBindingPower
	PrimaryBindingPower // Primary expressions, such as literals
)

var BindingPowerMap = map[lexer.TokenType]BindingPower{
	lexer.Comma: CommaBindingPower,

	lexer.ColonEqual: AssignBindingPower,
	lexer.EqualSign:  AssignBindingPower,
	lexer.PlusEqual:  AssignBindingPower,
	lexer.MinusEqual: AssignBindingPower,

	lexer.LessThan:       RelationalBindingPower,
	lexer.GreaterThan:    RelationalBindingPower,
	lexer.LessEqualTo:    RelationalBindingPower,
	lexer.GreaterEqualTo: RelationalBindingPower,
	lexer.Equals:         RelationalBindingPower,
	lexer.NotEqual:       RelationalBindingPower,

	lexer.LogicalAnd: LogicalBindingPower,
	lexer.LogicalOr:  LogicalBindingPower,

	lexer.Plus:  AdditiveBindingPower,
	lexer.Minus: AdditiveBindingPower,

	lexer.Times:    MultiplicativeBindingPower,
	lexer.Divide:   MultiplicativeBindingPower,
	lexer.Modulo:   MultiplicativeBindingPower,
	lexer.Exponent: MultiplicativeBindingPower,

	lexer.String:     PrimaryBindingPower,
	lexer.Numeric:    PrimaryBindingPower,
	lexer.Boolean:    PrimaryBindingPower,
	lexer.Identifier: PrimaryBindingPower,
}
