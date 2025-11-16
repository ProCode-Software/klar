package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
)

type CompileError interface {
	error
	GetRange() ranges.Range
	GetCode() ErrorCode
	GetHints() []Hint
	GetFile() string
	GetDetails() []Detail
	GetLabel() string
	GetHighlights() []Highlight
}

//go:generate stringer -type=ErrorCode
type (
	Ranges      = []ranges.Range
	ErrorParams map[string]any
	ErrorCode   int
	Highlight   struct {
		Range   ranges.Range
		Message string
	}
	Detail struct {
		File string
		Highlight
	}
	Hint struct {
		Message string
	}
)

const (
	SyntaxErrorPrefix ErrorCode = iota * 100
	WarningPrefix
	TypeErrorPrefix
	ReferenceErrorPrefix
	ImplementationErrorPrefix
)

const ErrTooManyErrors ErrorCode = -1 // Too many errors

type BaseError struct {
	ErrorCode  ErrorCode
	File       string
	Range      ranges.Range
	Message    string      // After underline
	Highlights []Highlight // Additional underline; same file
	Details    []Detail    // May be in different files
	Hints      []string
	Params     ErrorParams
}

func hintf(hints []Hint, f string, a []any) []Hint {
	if len(a) == 0 {
		hints = append(hints, Hint{Message: f})
		return hints
	}
	return append(hints, Hint{Message: fmt.Sprintf(f, a...)})
}
