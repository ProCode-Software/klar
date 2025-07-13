package errors

import "github.com/ProCode-Software/klar/internal/ranges"

type Warning struct {
	ErrorCode ErrorCode
	Name      string
	File      string
	Range     ranges.Range
	Params    ErrorParams
	Hints     []string
	Details   []Detail
}

const (
	_ ErrorCode = WarningPrefix + iota

	WarnNotEqualOr  // Always true: x != y || x != z
	WarnEqualAnd    // Never true: x == y && x == z
	WarnUnreachable // Unreachable code (after panic)
	WarnUnused      // Unused value
)

func (w Warning) Error() string {
	switch w.ErrorCode {
	default:
		return "Warning: " + w.ErrorCode.String()
	case WarnNotEqualOr:
		return "Warning: This logical expression is always true: did you mean to use '&&' to compare inequality?"
	}
}
