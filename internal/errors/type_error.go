package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
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
	ErrTypeCycle         // Circular type reference
	ErrInvalidRestType   // Rest type used where it is not supposed to
	ErrInvalidRestExpr   // Rest expression used where it is not supposed to
	ErrNoGenerics        // Only builtin types are generic
	ErrVariadicLast      // Variadic param must be last
	ErrWrongTypeParamLen // Wrong number of generic params

	ErrInvalidEnumValue     // Enum value must be literal string or number
	ErrCannotInferEnumValue // Explicit enum value required for strings

	ErrInheritNonStructOrIntf // In type declaration, can only inherit from struct or interface
	ErrConflictingInherit     // Name collision in struct inheritance
)

type TypeError struct {
	Name                  string
	Range                 ranges.Range
	ErrorCode             ErrorCode
	Params                ErrorParams
	ExpectedType, GotType types.Type
	Hints                 []string
}

func (e *TypeError) SetParam(key string, value any) TypeError {
	e.Params[key] = value
	return *e
}

func param[T any](params ErrorParams, key string) T {
	return params[key].(T)
}

func (e TypeError) Error() string {
	var (
		expType = e.ExpectedType
		gotType = e.GotType
		p       = e.Params
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
		types := p["types"].([2]string)
		switch {
		// Infinite size struct or interface:
		// 	type A { value: A }
		case p["mode"] == 1:
			return fmt.Sprintf(
				"TypeError: Invalid recursive type in %s",
				QuoteString(types[0]),
			)
		// Defined in terms of itself: type A = A
		case p["isSelf"]:
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
	case ErrConflictingInherit:
		return fmt.Sprintf(
			"TypeError: Field %s inherited from _ conflicts with already inherited field from _",
			QuoteString(e.Name),
		)
	case ErrVariadicLast:
		return "TypeError: Variadic parameter must be the last parameter"
	case ErrInvalidRestType:
		return "TypeError: Rest type isn't allowed here"
	case ErrNoGenerics:
		return fmt.Sprintf("TypeError: Type '%s' is not generic", p["type"])
	case ErrWrongTypeParamLen:
		return fmt.Sprintf(
			"TypeError: Expected between %d and %d type parameters, but found %d",
			param[int](p, "min"), param[int](p, "max"), param[int](p, "got"),
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

func NewTypeErr(code ErrorCode, rang ranges.Range, params ErrorParams) TypeError {
	return TypeError{
		ErrorCode: code,
		Range:     rang,
		Params:    params,
	}
}

func NodeTypeErr(code ErrorCode, node ast.Node, params ErrorParams) TypeError {
	return TypeError{
		ErrorCode: code,
		Range:     node.Base().Range,
		Params:    params,
	}
}
