package klarerrs

import "fmt"

const (
	_ Code = WarningPrefix + iota

	WarnNotEqualOr      // Always true: x != y || x != z
	WarnEqualAnd        // Never true: x == y && x == z
	WarnUnreachable     // Unreachable code (after panic)
	WarnUnused          // Unused value
	WarnOverloadResolve // Couldn't find an uncontested best overload for the given parameters
)

func (e *Error) handleWarning() string {
	switch e.Code {
	default:
		e.noMessage()
		return ""
	case WarnUnused:
		kind := e.StringParam("kind")
		e.Hintf("Delete it or prefix the name with '_' (e.g. '_%s')", e.Name)
		return fmt.Sprintf("%s %s is never used", kind, Quote(e.Name))
	case WarnNotEqualOr:
		return "This logical expression is always true: did you mean to use '&&' to compare inequality?"
	case WarnOverloadResolve:
		return "I struggled finding the best overload for these parameters"
	}
}
