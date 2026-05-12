package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

const (
	_ ErrorCode = TypeErrorPrefix + iota

	ErrAliasSelfType        // Method self type can't be a type alias
	ErrUnsupportedSelfType  // Self type doesn't support methods
	ErrUnsupportedInitType  // Initializer target doesn't support initializers
	ErrInvalidInheritedType // Invalid inherited type in declaration

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

type TypeError struct {
	File                  string
	ErrorCode             ErrorCode
	Name                  string
	Range                 ranges.Range
	ExpectedType, GotType types.Type
	Params                ErrorParams
	Label                 string
	Details               []Detail
	Hints                 []Hint
	Highlights            []Highlight
}

func (e *TypeError) SetParam(key string, value any) *TypeError {
	if e.Params == nil {
		e.Params = make(ErrorParams)
	}
	e.Params[key] = value
	return e
}

func param[T any](params ErrorParams, key string) T {
	return params[key].(T)
}

func (e *TypeError) Error() string {
	var (
		expType = e.ExpectedType
		gotType = e.GotType
		name    = e.Name
		p       = e.Params
	)
	switch e.ErrorCode {
	default:
		return e.GetCode().String()

	case ErrAliasSelfType:
		if param[bool](e.Params, "initializer") {
			return "The target of an initializer can't be a type alias"
		}
		return "A method's self type can't be a type alias"
	case ErrUnsupportedSelfType:
		return "You can only declare methods on struct and enum types"
	case ErrUnsupportedInitType:
		return "You can only create initalizers for struct and enum types"
	case ErrInvalidInheritedType:
		allowed := param[string](p, "allowedTypes")
		kind := param[string](p, "kind")
		return kind + " can only inherit " + allowed

	// OLD ERRORS
	// =======
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
				Quote(types[0]),
			)
		// Defined in terms of itself: type A = A
		case p["isSelf"]:
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
		if meth := param[*types.Function](p, "method"); meth != nil {
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
		return fmt.Sprintf("TypeError: Type '%s' is not generic", p["type"])
	case ErrWrongTypeParamLen:
		return fmt.Sprintf(
			"TypeError: Expected between %d and %d type parameters, but got %d",
			param[int](p, "min"), param[int](p, "max"), param[int](p, "got"),
		)
	case ErrNonStructReceiver:
		return fmt.Sprintf(
			"TypeError: Can't define method on %s: Type %[1]s is %s and is not a struct",
			Quote(e.Name),
			QuoteType(e.GotType),
		)
	case ErrOverloadExists:
		return fmt.Sprintf(
			"TypeError: Overload %s was already defined at %s",
			Quote(name), param[ranges.Range](p, "origPos").Start,
		)
	case ErrMismatchedDistrib:
		return fmt.Sprintf(
			"TypeError: Operands in distributive expression must be the same type: found mismatched %s and %s",
			QuoteType(expType), QuoteType(gotType),
		)
	case ErrUncomparableTypes:
		return fmt.Sprintf("TypeError: Can't compare type %s with %s operator",
			QuoteType(gotType),
			FormatTokenType(param[lexer.TokenType](p, "operator")),
		)
	}
}

func TypeMismatch(expType, gotType types.Type, rang ranges.Range) *TypeError {
	return &TypeError{
		ErrorCode:    ErrTypeMismatch,
		ExpectedType: expType,
		GotType:      gotType,
		Range:        rang,
	}
}

func NamedTypeError(code ErrorCode, name string, rang ranges.Range) *TypeError {
	return &TypeError{
		ErrorCode: code,
		Name:      name,
		Range:     rang,
	}
}

func RangedTypeError(code ErrorCode, rang ranges.Range, params ErrorParams) *TypeError {
	return &TypeError{
		ErrorCode: code,
		Range:     rang,
		Params:    params,
	}
}

func NodeTypeError(code ErrorCode, node ast.Node, params ErrorParams) *TypeError {
	return &TypeError{
		ErrorCode: code,
		Range:     node.GetRange(),
		Params:    params,
	}
}

func OperatorTypeError(rang ranges.Range, l, r types.Type, op lexer.TokenType) *TypeError {
	return &TypeError{
		ErrorCode:    ErrMismatchedOperands,
		Range:        rang,
		ExpectedType: l,
		GotType:      r,
		Params:       ErrorParams{"operator": op},
	}
}
