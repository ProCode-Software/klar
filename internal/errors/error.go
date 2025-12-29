package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
)

type CompileError interface {
	error
	GetName() string
	GetMessage() string
	GetRange() ranges.Range
	GetCode() ErrorCode
	GetHints() []Hint
	GetFile() string
	GetDetails() []Detail
	GetLabel() string
	GetHighlights() []Highlight
	addDetail(d Detail)
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
	ModuleErrorPrefix
	ImplementationErrorPrefix
)

const ErrTooManyErrors ErrorCode = -1 // Too many errors

func hintf(hints []Hint, f string, a []any) []Hint {
	if len(a) == 0 {
		hints = append(hints, Hint{Message: f})
		return hints
	}
	return append(hints, Hint{Message: fmt.Sprintf(f, a...)})
}

func TooManyErrors() *ParseError {
	return &ParseError{ErrorCode: ErrTooManyErrors}
}

func AddDetail(err CompileError, file string, rang ranges.Range, msg string) {
	err.addDetail(Detail{file, Highlight{rang, msg}})
}
