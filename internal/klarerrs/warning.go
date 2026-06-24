package klarerrs

import "fmt"

const (
	_ Code = WarningPrefix + iota

	WarnNotEqualOr  // Always true: x != y || x != z
	WarnEqualAnd    // Never true: x == y && x == z
	WarnUnreachable // Unreachable code (after panic)
	WarnUnused      // Unused value
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
		return "Warning: This logical expression is always true: did you mean to use '&&' to compare inequality?"
	}
}
