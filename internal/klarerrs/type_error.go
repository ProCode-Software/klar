package klarerrs

import (
	"fmt"
)

const (
	_ Code = TypeErrorPrefix + iota

	ErrTypeMismatch   // Type mismatch
	ErrUnwrapRequired // Optional/Result type must be unwrapped before use

	// Declaration ====

	ErrAliasSelfType          // Method self type can't be a type alias
	ErrUnsupportedSelfType    // Self type doesn't support methods
	ErrUnsupportedInitType    // Initializer target doesn't support initializers
	ErrInvalidInheritedType   // Invalid inherited type in declaration
	ErrAliasAndMethodSameName // Method and alias have the same name
	ErrFieldAndMethodSameName // Field and method have the same name
	ErrEnumSameValue          // Enum value must be unique
	ErrCantInferStringEnum    // Can't infer string enum value
	ErrUnknownAttribute       // Unknown attribute
	ErrGenericTypeAlias       // Type alias cannot be a generic type
	ErrDepCycle               // Circular type reference
	ErrMismatchTupleDestruct  // Number of destructured tuple items on left > right
	ErrTupleRestDestruct      // A rest in a tuple destructure must give the target at least 2 items
	ErrOverloadReturnMismatch // Overloads must return the same type
	ErrInvalidInitReturn      // Initializer for T must return T | T? | Result<T>
	ErrInvalidListInitReturn  // Initializer for List must return a list (List | List? | Result<List>)
	ErrMissingReturn          // Function doesn't return Nothing but contains no return statements

	// Type expression ====

	ErrNotAType        // Variable or function used in type context
	ErrTypeAsValue     // Type used as a value
	ErrInvalidRestType // Rest type used outside of function parameter

	// Statement ====

	ErrNotIterable         // Type isn't iterable (can be used in a 'for' loop)
	ErrNonBoolWhileCond    // Condition in 'while' statement must be type Bool
	ErrOver2LoopVars       // Can't declare more than 2 loop variables in a 'for' loop
	ErrMultipleIntIterVars // Only 1 variable allowed when iterating over Int
	ErrAssignToConst       // Attempted reassignment to constant reference

	// Literal ====

	ErrUnknownStructShorthand // Can't determine type of struct from shorthand (`.(...)`)
	ErrUnknownEnumShorthand   // Can't determine type of enum from shorthand (`.key`)

	// Expression ====

	ErrInvalidRangeType      // Can't range over this type
	ErrStepWithStringRange   // Step isn't allowed with String range
	ErrNonConstStringRange   // Range bounds must be constants when ranging over String
	ErrOpenStringRange       // '..<' not allowed with range over String
	ErrNonLetterStringRange  // Bounds of range over String must be a letter or digit
	ErrMultiCharStringRange  // Bounds of range over String must be a single character
	ErrInvalidIndexType      // Can't index this type
	ErrNonNumericIndex       // Index for list/String/tuple must be Int
	ErrInvalidMapIndex       // Map must be indexed with its key type
	ErrFieldNotFound         // Field not found
	ErrInvalidComputedIndex  // Computed index not supported for this type
	ErrDotIndexRequired      // Dot index required to index this type instead of computed String index
	ErrNegateNonNumeric      // Negate '-' operator only supported on Int and Float
	ErrNonBoolLogicalOperand // Operand to '||', '&&', and '!' operator must be Bool
	ErrInvalidArithType      // Operand for arithmetic operation must be numeric
	ErrIntTimesString        // Should be String * Int, in that exact order
	ErrInvalidStringMult     // String must be multiplied by Int
	ErrNonBoolLogical        // Operands in logical expression must be boolean
	ErrInvalidInOperand      // Right-hand side of 'in' operator must be a list or map
	ErrUntypedEmptyList      // Can't infer type of empty list
	ErrUntypedNil            // 'nil' requires a type (explicit type at assignment)
	ErrNothingAsValue        // Function returning Nothing can't be used as a value
	ErrNonResultInTry        // Expression after 'try' must be a Result
	ErrInvalidAssertType     // Type being asserted must be a result or optional
	ErrNotAFunction          // Can't call a non-function or initializer
	ErrGenericParamsRequired // Reference to generic type requires params
	ErrNonGenericType        // Generics passed to type that doesn't accept any
	ErrInvalidGenericCount   // Too few/many generic parameters passed
)

func (e *Error) handleTypeError() string {
	name := Quote(e.Name)
	info, _ := e.Info.(TypeErrorInfo)
	switch e.Code {
	default:
		e.noMessage()
		return ""

	case ErrTypeMismatch:
		return "Type mismatch: expected type " + Quote(info.ExpectedType) +
			", but this has type " + Quote(info.GotType)
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
	case ErrAliasAndMethodSameName:
		return "Method alias " + Quote(name) + " has the same name as another method"
	case ErrEnumSameValue:
		key := e.StringParam("key")
		otherKey := e.StringParam("otherKey")
		return fmt.Sprintf(
			"Enum item %s has the same value as %s",
			Quote(key), Quote(otherKey),
		)
	case ErrUnknownAttribute:
		return "I don't recognize the " + name + " attribute"
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
	case ErrTypeAsValue:
		return "Can't use type " + Quote(e.Name) + " as a value"
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
	case ErrInvalidIndexType:
		return "Can't index type " + Quote(info.GotType)
	case ErrNonNumericIndex:
		return "Type " + Quote(info.ExpectedType) + " must be indexed with Int"
	case ErrInvalidMapIndex:
		return "Map with type " + Quote(e.Name) +
			" must be indexed with the same type as its keys, which is " +
			Quote(info.ExpectedType)
	case ErrFieldNotFound:
		return "Can't find a field or method named " + Quote(e.Name) +
			" on type " + Quote(e.StringParam("type"))
	case ErrInvalidComputedIndex:
		return "Can't use type " + Quote(info.GotType) + " to index type " + Quote(e.Name)
	case ErrDotIndexRequired:
		return "Type " + Quote(e.Name) + "'s fields must be accessed via a dot-index"
	case ErrInvalidArithType:
		return "The " + Quote(e.Name) + " operator is only supported with numeric operands"
	case ErrNonBoolLogicalOperand:
		return "Logical operator " + Quote(e.Name) + " requires Bool or optional operands"
	case ErrInvalidStringMult:
		return "Type String can only be multiplied by Int"
	case ErrFieldAndMethodSameName:
		return "Method " + Quote(e.Name) + " has the same name as a field on type " +
			Quote(e.StringParam("type"))
	case ErrOverloadReturnMismatch:
		return "Overloads of " + Quote(e.Name) + " must return the same type, " +
			Quote(info.ExpectedType) + ", but this one returns " + Quote(info.GotType)
	case ErrInvalidInitReturn:
		return fmt.Sprintf(
			"An initializer for '%s' must return '%[1]s', 'Result<%[1]s>', or '%[1]s?'",
			e.Name,
		)
	case ErrInvalidListInitReturn:
		return "An initializer for 'List' must return a list, possibly as an optional or result"
	case ErrMissingReturn:
		return "This function is supposed to return " + Quote(e.Name) + ", but the body contains no 'return' statements"
	case ErrUntypedEmptyList:
		return "I can't determine the item type of this empty list"
	case ErrUntypedNil:
		return "'nil' requires an explicit type at assignment"
	case ErrNothingAsValue:
		return "This function returns Nothing and can't be used as a value"
	case ErrNonResultInTry:
		return "The expression after 'try' must be a Result"
	case ErrInvalidAssertType:
		return "The expression before '!!' must be a Result or optional"
	case ErrNotAFunction:
		return "Type " + Quote(e.Name) + " isn't a function and can't be called"
	}
}
