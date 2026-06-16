package klarerrs

import (
	"fmt"
)

const (
	_ Code = ReferenceErrorPrefix + iota

	ErrUndefined
	ErrEnumUndefined   // Enum item doesn't exist
	ErrEnumCycle       // Enum items refer to each other
	ErrExportUndefined // Item doesn't exist in module
	ErrNotExported     // Can't import an exported object
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
	case ErrExportUndefined:
		module := e.StringParam("module")
		return fmt.Sprintf("Can't find %s in module %s", name, Quote(module))
	case ErrNotExported:
		return name + " from module " + Quote(e.StringParam("module")) + " isn't public"
	}
}
