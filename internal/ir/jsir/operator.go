package jsir

//go:generate go tool stringer -type=Operator -linecomment

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Expressions_and_operators
type Operator uint8

const (
	_ Operator = iota

	// Comparison

	OpEqual          // ==
	OpNotEqual       // !=
	OpStrictEqual    // ===
	OpStrictNotEqual // !==
	OpGreaterThan    // >
	OpLessThan       // <
	OpGreaterEqual   // >=
	OpLessEqual      // <=

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
