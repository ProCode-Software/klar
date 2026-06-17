package klarerrs

import (
	"fmt"
)

const (
	_ Code = TypeErrorPrefix + iota

	ErrAliasSelfType          // Method self type can't be a type alias
	ErrUnsupportedSelfType    // Self type doesn't support methods
	ErrUnsupportedInitType    // Initializer target doesn't support initializers
	ErrInvalidInheritedType   // Invalid inherited type in declaration
	ErrAliasAndMethodSameName // Method and alias have the same name
	ErrEnumSameValue          // Enum value must be unique
	ErrCantInferStringEnum    // Can't infer string enum value

	// Old errors

	ErrUntypedNil       // nil requires contextual type
	ErrUntypedEmptyList // Can't infer type from empty list
	ErrUntypedEnum      // Shorthand enum syntax without enum type

	ErrUncheckedOptional // Required to check if optional is nil
	ErrUncheckedResult   // Required to check Result for error
	ErrInvalidRestType   // Rest type used where it is not supposed to
	ErrInvalidRestExpr   // Rest expression used where it is not supposed to
	ErrVariadicLast      // Variadic param must be last

	ErrTypeCycle         // Circular type reference
	ErrNoGenerics        // Only builtin types are generic
	ErrWrongTypeParamLen // Wrong number of generic params
	ErrInvalidEnumValue  // Enum value must be literal string or number

	ErrInheritNonStructOrIntf // In type declaration, can only inherit from struct or interface
	ErrConflictingInherit     // Name collision in struct inheritance
	ErrNonStructReceiver      // Defining method on non-struct type
	ErrOverloadExists         // Overload already defined

	ErrAssignToConst   // Attempted reassignment to constant reference
	ErrTypeMismatch    // Type mismatch
	ErrWrongAssignType // Wrong type for assignment

	ErrNonBoolLogical     // Operands in logical expression must be boolean
	ErrMismatchedOperands // Operands don't match
	ErrMismatchedDistrib  // Distributive operands must be the same type
	ErrUncomparableTypes  // Uncomparable types in relational expression
	ErrIntTimesString     // Wrong side for string multiplication
	ErrInvalidOperation   // Operands are same type, but arithmetic not allowed on type
)

func (e *Error) handleTypeError() string {
	switch e.Code {
	default:
		e.noMessage()
		return ""

	case ErrAliasSelfType:
		if e.BoolParam("initializer") {
			return "The target of an initializer can't be a type alias"
		}
		return "A method's self type can't be a type alias"
	case ErrUnsupportedSelfType:
		return "You can only declare methods on struct and enum types"
	case ErrUnsupportedInitType:
		return "You can only create initializers for struct and enum types"
	case ErrInvalidInheritedType:
		allowed := e.StringParam("allowedTypes")
		kind := e.StringParam("kind")
		return kind + " can only inherit " + allowed
	case ErrEnumSameValue:
		key := e.StringParam("key")
		otherKey := e.StringParam("otherKey")
		return fmt.Sprintf(
			"Enum item %s has the same value as %s",
			Quote(key), Quote(otherKey),
		)

		// OLD ERRORS
		// =======
		/*
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
		*/
	}
}
