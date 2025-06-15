package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

const (
	_ ErrorCode = TypeErrorPrefix + iota

	ErrUntypedNil        // nil requires contextual type
	ErrUntypedEmptyList  // Can't infer type from empty list
	ErrAssignToConst     // Attempted reassignment to constant reference
	ErrUncheckedOptional // Required to check if optional is nil
	ErrUncheckedResult   // Required to check Result for error
	ErrUnusedLiteral     // Unused literal expression statement
	ErrUnusedLastLit     // Same as above, but last statement in block
	ErrTypeMismatch      // Type mismatch
	ErrRecursiveType     // Recursive type that is not union, array, or optional in struct
	ErrTypeCycle         // Circular type reference

	ErrInvalidEnumValue     // Enum value must be literal string or number
	ErrCannotInferEnumValue // Explicit enum value required for strings
)

type TypeError struct {
	Name                  string
	Range                 ranges.Range
	ErrorCode             ErrorCode
	Params                ErrorParams
	ExpectedType, GotType types.Type
	Hints                 []string
}

func (e TypeError) Error() string {
	var (
		expType = e.ExpectedType
		gotType = e.GotType
		params  = e.Params
	)
	switch e.ErrorCode {
	default:
		return "TypeError: " + e.Code().String()
	case ErrTypeMismatch:
		return fmt.Sprintf("TypeError: This is supposed to be a %T, not %T",
			expType, gotType,
		)
	case ErrInvalidEnumValue:
		return "TypeError: Enum values can only be 'String', 'Int', or 'Float'"
	case ErrTypeCycle:
		types := params["types"].([2]string)
		switch {
		// Infinite size struct or interface:
		// 	type A { value: A }
		case params["mode"] == 1:
			return fmt.Sprintf(
				"TypeError: Invalid recursive type in %s",
				QuoteString(types[0]),
			)
		// Defined in terms of itself: type A = A
		case params["isSelf"]:
			return fmt.Sprintf(
				"TypeError: Type %s references itself",
				QuoteString(types[0]),
			)
		}
		// Other cycle
		return fmt.Sprintf(
			"TypeError: Type cycle: %s and %s recursively reference each other",
			QuoteString(types[0]), QuoteString(types[1]),
		)
	}
}

func TypeMismatch(expType, gotType types.Type, rang ranges.Range) TypeError {
	return TypeError{
		ErrorCode:    ErrTypeMismatch,
		ExpectedType: expType,
		GotType:      gotType,
		Range:        rang,
	}
}

func NamedTypeError(code ErrorCode, name string, rang ranges.Range) TypeError {
	return TypeError{
		ErrorCode: code,
		Name:      name,
		Range:     rang,
	}
}
