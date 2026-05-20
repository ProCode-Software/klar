package klarerrs

import (
	"fmt"
)

const (
	_ Code = ReferenceErrorPrefix + iota

	ErrUndefined
	ErrEnumUndefined // Enum item doesn't exist
	ErrEnumCycle     // Enum items refer to each other
)

func (e *Error) handleReferenceError() string {
	name := Quote(e.Name)
	switch e.Code {
	default:
		e.noMessage()
		return ""
	case ErrEnumUndefined:
		return fmt.Sprintf(
			"Can't find item %s in enum %s",
			name,
			Quote(e.StringParam("enumName")),
		)
	case ErrUndefined:
		return fmt.Sprintf("Can't find %s in scope", name)
	case ErrEnumCycle:
		return ""
	}
}
