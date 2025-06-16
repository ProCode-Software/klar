package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	_ ErrorCode = ReferenceErrorPrefix + iota

	ErrVarUndefined  // Variable doesn't exist
	ErrEnumUndefined // Enum item doesn't exist
	ErrTypeUndefined // Type doesn't exist
)

type ReferenceError struct {
	Name      string
	ErrorCode ErrorCode
	Range     ranges.Range
	Hints     []string
	Params    ErrorParams
}

func (e *ReferenceError) SetParam(key string, value any) ReferenceError {
	if e.Params == nil {
		e.Params = make(ErrorParams)
	}
	e.Params[key] = value
	return *e
}

func (e ReferenceError) At() lexer.Position    { return e.Range.Start }
func (e ReferenceError) AtRange() ranges.Range { return e.Range }
func (e ReferenceError) Code() ErrorCode       { return e.ErrorCode }
func (e ReferenceError) GetHints() []string    { return e.Hints }

func (e ReferenceError) Error() string {
	switch e.ErrorCode {
	default:
		return "ReferenceError: " + e.ErrorCode.String()
	case ErrEnumUndefined:
		return fmt.Sprintf(
			"ReferenceError: Can't find item %s in enum %s",
			QuoteString(e.Name),
			QuoteString(param[string](e.Params, "enumName")),
		)
	case ErrTypeUndefined:
		return fmt.Sprintf("ReferenceError: Can't find type %s in scope",
			QuoteString(e.Name),
		)
	}
}

func Undefined(code ErrorCode, name string, rang ranges.Range) ReferenceError {
	return ReferenceError{
		ErrorCode: code,
		Name:      name,
		Range:     rang,
	}
}
