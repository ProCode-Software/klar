package jsir

//go:generate go tool stringer -type=Operator -linecomment

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Expressions_and_operators
type Operator uint8

const (
	_ Operator = iota

	// Relational/Comparison

	OpEqual          // ==
	OpNotEqual       // !=
	OpStrictEqual    // ===
	OpStrictNotEqual // !==
	OpGreaterThan    // >
	OpLessThan       // <
	OpGreaterEqual   // >=
	OpLessEqual      // <=
	OpIn             // in
	OpInstanceof     // instanceof

	// Arithmetic

	OpPlus      // +
	OpMinus     // -
	OpTimes     // *
	OpDivide    // /
	OpRemainder // %
	OpIncrement // ++
	OpDecrement // --
	OpExponent  // **

	// Bitwise

	OpBitwiseAnd  // &
	OpBitwiseOr   // |
	OpBitwiseXOr  // ^
	OpBitwiseNot  // ~
	OpShiftLeft   // <<
	OpShiftRight  // >>
	OpUShiftRight // >>>

	// Logical

	OpLogicalAnd   // &&
	OpLogicalOr    // ||
	OpNullCoalesce // ??
	OpLogicalNot   // !

	// Assignment

	OpPlusEqual         // +=
	OpMinusEqual        // -=
	OpTimesEqual        // *=
	OpDivideEqual       // /=
	OpRemainderEqual    // %=
	OpExponentEqual     // **=
	OpShiftLeftEqual    // <<=
	OpShiftRightEqual   // >>=
	OpUShiftRightEqual  // >>>=
	OpBitwiseAndEqual   // &=
	OpBitwiseOrEqual    // |=
	OpBitwiseXOrEqual   // ^=
	OpLogicalAndEqual   // &&=
	OpLogicalOrEqual    // ||=
	OpNullCoalesceEqual // ??=

	// Unary

	OpAwait     // await
	OpDelete    // delete
	OpVoid      // void
	OpTypeof    // typeof
	OpNew       // new
	OpYield     // yield
	OpYieldStar // yield*
)

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Operator_precedence
func (op Operator) Precedence() int {
	return 0
}

type Precedencer interface {
	Precedence() int
}

// ShouldAddSpace reports whether spaces should be written around the operator.
// ShouldAddSpace assumes the operator is used in a unary context. If the operator
// isn't a unary operator, it returns false.
func (op Operator) ShouldAddSpace() bool {
	switch op {
	case OpBitwiseNot, OpIncrement, OpDecrement, OpLogicalNot, OpPlus, OpMinus:
		return true
	}
	return false
}
