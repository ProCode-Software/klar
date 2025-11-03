package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	_ ErrorCode = ReferenceErrorPrefix + iota

	ErrVarUndefined  // Variable doesn't exist
	ErrEnumUndefined // Enum item doesn't exist
	ErrTypeUndefined // Type doesn't exist
	ErrEnumCycle     // Enum items refer to each other
)

type CycleItem struct {
	Name     string
	Position ranges.Range
}

type ReferenceError struct {
	File      string
	Name      string
	ErrorCode ErrorCode
	Range     ranges.Range
	Details   []Detail
	Hints     []string
	Params    ErrorParams
}

func (e *ReferenceError) SetParam(key string, value any) *ReferenceError {
	if e.Params == nil {
		e.Params = make(ErrorParams)
	}
	e.Params[key] = value
	return e
}

func (e *ReferenceError) Error() string {
	name := Quote(e.Name)
	switch e.ErrorCode {
	default:
		return "ReferenceError: " + e.ErrorCode.String()
	case ErrEnumUndefined:
		return fmt.Sprintf(
			"ReferenceError: Can't find item %s in enum %s",
			name,
			Quote(param[string](e.Params, "enumName")),
		)
	case ErrTypeUndefined:
		return fmt.Sprintf("ReferenceError: Can't find type %s in scope", name)
	case ErrVarUndefined:
		return fmt.Sprintf("ReferenceError: Can't find %s in scope", name)
	case ErrEnumCycle:
		cycle := param[[]CycleItem](e.Params, "cycle")
		return fmt.Sprintf(
			"ReferenceError: Enum items %s and %s recursively reference each other",
			name,
			Quote(cycle[len(cycle)-1].Name),
		)
	}
}

func Undefined(code ErrorCode, name string, rang ranges.Range) *ReferenceError {
	return &ReferenceError{
		ErrorCode: code,
		Name:      name,
		Range:     rang,
	}
}
