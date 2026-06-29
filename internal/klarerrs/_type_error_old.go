package klarerrs

// Error codes for the previous type checker implementation. This file
// exists for reference only.

const (
	ErrUntypedNil             = iota // nil requires contextual type
	ErrUntypedEmptyList              // Can't infer type from empty list
	ErrUntypedEnum                   // Shorthand enum syntax without enum type
	ErrUncheckedOptional             // Required to check if optional is nil
	ErrUncheckedResult               // Required to check Result for error
	ErrInvalidRestExpr               // Rest expression used where it is not supposed to
	ErrNoGenerics                    // Only builtin types are generic
	ErrWrongTypeParamLen             // Wrong number of generic params
	ErrInvalidEnumValue              // Enum value must be literal string or number
	ErrInheritNonStructOrIntf        // In type declaration, can only inherit from struct or interface
	ErrConflictingInherit            // Name collision in struct inheritance
	ErrNonStructReceiver             // Defining method on non-struct type
	ErrOverloadExists                // Overload already defined
	ErrWrongAssignType               // Wrong type for assignment
	ErrMismatchedOperands            // Operands don't match
	ErrMismatchedDistrib             // Distributive operands must be the same type
	ErrUncomparableTypes             // Uncomparable types in relational expression
	ErrInvalidOperation              // Operands are same type, but arithmetic not allowed on type
)

func (e *Error) _() string {
	switch e.Code {
	case ErrTypeMismatch:
		return fmt.Sprintf(
			"TypeError: This is supposed to be a %T, not %T",
			expType, gotType,
		)
	case ErrInvalidEnumValue:
		return "TypeError: Enum values can only be 'String', 'Int', or 'Float'"
	case ErrTypeCycle:
		types := e.Params["types"].([2]string)
		switch {
		// Infinite size struct or interface:
		// 	type A { value: A }
		case e.Params["mode"] == 1:
			return fmt.Sprintf(
				"TypeError: Invalid recursive type in %s",
				Quote(types[0]),
			)
		// Defined in terms of itself: type A = A
		case e.BoolParam("isSelf"):
			return fmt.Sprintf(
				"TypeError: Type %s references itself",
				Quote(types[0]),
			)
		}
		// Other cycle
		return fmt.Sprintf(
			"TypeError: Type cycle: %s and %s recursively reference each other",
			Quote(types[0]), Quote(types[1]),
		)
	case ErrConflictingInherit:
		if meth, _ := e.Params["method"].(*types.Function); meth != nil {
			return fmt.Sprintf(
				"TypeError: Method %s inherited from _ conflicts with already inherited method from _",
				Quote(meth.StringNamed(name)),
			)
		}
		return fmt.Sprintf(
			"TypeError: Field %s inherited from _ conflicts with already inherited field from _",
			Quote(name),
		)
	case ErrVariadicLast:
		return "TypeError: Variadic parameter must be the last parameter"
	case ErrInvalidRestType:
		return "TypeError: Rest type isn't allowed here"
	case ErrNoGenerics:
		return fmt.Sprintf("TypeError: Type '%s' is not generic", e.Params["type"])
	case ErrWrongTypeParamLen:
		return fmt.Sprintf(
			"TypeError: Expected between %d and %d type parameters, but got %d",
			e.IntParam("min"), e.IntParam("max"), e.IntParam("got"),
		)
	case ErrNonStructReceiver:
		return fmt.Sprintf(
			"TypeError: Can't define method on %s: Type %[1]s is %s and is not a struct",
			Quote(e.Name),
			QuoteType(gotType),
		)
	case ErrOverloadExists:
		return fmt.Sprintf(
			"TypeError: Overload %s was already defined at %s",
			Quote(name), e.Params["origPos"].(ranges.Range).Start,
		)
	case ErrMismatchedDistrib:
		return fmt.Sprintf(
			"TypeError: Operands in distributive expression must be the same type: found mismatched %s and %s",
			QuoteType(expType), QuoteType(gotType),
		)
	case ErrUncomparableTypes:
		return fmt.Sprintf(
			"TypeError: Can't compare type %s with %s operator",
			QuoteType(gotType),
			FormatTokenType(e.TokenTypeParam("operator")),
		)
	}
}
