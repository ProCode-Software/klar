package klarerrs

import (
	"fmt"
)

const (
	_ Code = TypeErrorPrefix + iota

	ErrTypeMismatch           // Type mismatch
	ErrAliasSelfType          // Method self type can't be a type alias
	ErrUnsupportedSelfType    // Self type doesn't support methods
	ErrUnsupportedInitType    // Initializer target doesn't support initializers
	ErrInvalidInheritedType   // Invalid inherited type in declaration
	ErrAliasAndMethodSameName // Method and alias have the same name
	ErrEnumSameValue          // Enum value must be unique
	ErrCantInferStringEnum    // Can't infer string enum value
	ErrAttributesNotAllowed   // Attributes are not allowed in this context (bootstrap)
	ErrUnknownAttribute       // Unknown attribute
	ErrGenericTypeAlias       // Type alias cannot be a generic type
	ErrDepCycle               // Circular type reference
	ErrNotAType               // Variable or function used in type context
	ErrInvalidRestType        // Rest type used outside of function parameter
	ErrNonBoolWhileCond       // Condition in 'while' statement must be type Bool
	ErrUnwrapRequired         // Optional/Result type must be unwrapped before use
	ErrNotIterable            // Type isn't iterable (can be used in a 'for' loop)
	ErrOver2LoopVars          // Can't declare more than 2 loop variables in a 'for' loop
	ErrMultipleIntIterVars    // Only 1 variable allowed when iterating over Int
	ErrTypeAsValue            // Type used as a value
	ErrUnknownStructShorthand // Can't determine type of struct from shorthand (`.(...)`)
	ErrUnknownEnumShorthand   // Can't determine type of enum from shorthand (`.key`)
	ErrInvalidRangeType       // Can't range over this type
	ErrStepWithStringRange    // Step isn't allowed with String range
	ErrNonConstStringRange    // Range bounds must be constants when ranging over String
	ErrOpenStringRange        // '..<' not allowed with range over String
	ErrNonLetterStringRange   // Bounds of range over String must be a letter or digit
	ErrMultiCharStringRange   // Bounds of range over String must be a single character
	ErrMismatchTupleDestruct  // Number of destructured tuple items on left > right
	ErrTupleRestDestruct      // A rest in a tuple destructure must give the target at least 2 items
	ErrInvalidTypeIndex       // Can't index this type
	ErrNegateNonNumeric       // Negate '-' operator only supported on Int and Float

	// Old errors. For reference only.

	ErrUntypedNil       // nil requires contextual type
	ErrUntypedEmptyList // Can't infer type from empty list
	ErrUntypedEnum      // Shorthand enum syntax without enum type

	ErrUncheckedOptional // Required to check if optional is nil
	ErrUncheckedResult   // Required to check Result for error
	ErrInvalidRestExpr   // Rest expression used where it is not supposed to
	ErrVariadicLast      // Variadic param must be last

	ErrNoGenerics        // Only builtin types are generic
	ErrWrongTypeParamLen // Wrong number of generic params
	ErrInvalidEnumValue  // Enum value must be literal string or number

	ErrInheritNonStructOrIntf // In type declaration, can only inherit from struct or interface
	ErrConflictingInherit     // Name collision in struct inheritance
	ErrNonStructReceiver      // Defining method on non-struct type
	ErrOverloadExists         // Overload already defined

	ErrAssignToConst   // Attempted reassignment to constant reference
	ErrWrongAssignType // Wrong type for assignment

	ErrNonBoolLogical     // Operands in logical expression must be boolean
	ErrMismatchedOperands // Operands don't match
	ErrMismatchedDistrib  // Distributive operands must be the same type
	ErrUncomparableTypes  // Uncomparable types in relational expression
	ErrIntTimesString     // Wrong side for string multiplication
	ErrInvalidOperation   // Operands are same type, but arithmetic not allowed on type
)

func (e *Error) handleTypeError() string {
	name := Quote(e.Name)
	info, _ := e.Info.(TypeErrorInfo)
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
	case ErrUnknownAttribute:
		return "I don't recognize the " + name + " attribute"
	case ErrAttributesNotAllowed:
		return "This module can't use attributes when being bootstrapped"
	case ErrGenericTypeAlias:
		return "The right-hand side of a type alias declaration can't be a generic"
	case ErrDepCycle:
		isTypeDecl := e.BoolParam("type")
		isSelf := e.BoolParam("self")
		if isSelf {
			return name + " is declared in terms of itself"
		}
		msg := name + " depends on itself in a cycle"
		if isTypeDecl {
			msg = "Type " + msg
		}
		return msg
	case ErrNotAType:
		actual := e.StringParam("kind")
		return name + " is " + WithA(actual) + ", not a type"
	case ErrInvalidRestType:
		return "'...' can only be used as a function parameter"
	case ErrNonBoolWhileCond:
		return "The condition in a 'while' statement has to be of type Bool"
	case ErrUnwrapRequired:
		kind := e.StringParam("kind")
		if kind == "" {
			kind = "Value"
		} else {
			kind += " value"
		}
		msg := kind + " of type " + Quote(info.GotType) + " must be unwrapped"
		if before := e.StringParam("before"); before != "" {
			msg += " " + before
		}
		return msg
	case ErrOver2LoopVars:
		return "Up to 2 variables can be declared in a 'for' loop"
	case ErrMultipleIntIterVars:
		return "Only 1 loop variable is allowed when iterating over an Int"
	case ErrNotIterable:
		if info.GotType == "Float" {
			e.Hint("Define a range or convert the value to an Int to iterate over it.")
			return "Can't iterate over a Float"
		}
		e.Hint("Iterable types include lists, Strings, Ints, and maps.")
		return "Can't iterate over type " + Quote(info.GotType)
	case ErrInvalidRangeType:
		e.Hint("You can range over String, Int, and Float")
		return "Can't range over type " + Quote(info.GotType)
	case ErrStepWithStringRange:
		return "A step can't be specified when ranging over type String"
	case ErrNonConstStringRange:
		return "The bounds of a range over String must be constants"
	case ErrOpenStringRange:
		return "'..<' can't be used when ranging over type String"
	case ErrMismatchTupleDestruct:
		return "The tuple on the right-hand side doesn't have enough values to assign to " +
			FormatThisWord(e.IntParam("remaining"), "destructured variable")
	case ErrTupleRestDestruct:
		name := "the target" /* Quote(e.Name) */
		return "A rest in a tuple destructure must give " + name + " at least 2 items"
	case ErrNegateNonNumeric:
		return "The expression after '-' must be a number"
	case ErrInvalidTypeIndex:
		return "Can't index type " + Quote(info.GotType)

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
