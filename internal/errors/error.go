package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
)

//go:generate ./
type CompileError interface {
	error
	GetRange() ranges.Range
	Code() ErrorCode
	GetHints() []string
	GetFile() string
	GetDetails() []Detail
	GetMessage() string
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
)

const (
	SyntaxErrorPrefix ErrorCode = iota * 100
	WarningPrefix
	TypeErrorPrefix
	ReferenceErrorPrefix
	ImplementationErrorPrefix
)

const ErrMaxErrors = -1 // Too many errors

func (e *ParseError) GetRange() ranges.Range { return e.Range }

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

func (err *BaseError) GetFile() string          { return err.File }
func (err *BaseError) GetDetails() []Highlight  { return err.Details }
func (err *BaseError) GetHighlight() *Highlight { return err.Highlight }
func (err *BaseError) GetMessage() string       { return err.Message }
func (err *BaseError) GetCode() ErrorCode       { return err.ErrorCode }
func (err *BaseError) GetHints() []string       { return err.Hints }
func (err *BaseError) GetRange() ranges.Range   { return err.Range }
func (err *BaseError) Hint(hint string)         { err.Hints = append(err.Hints, hint) }
func (e *BaseError) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// SyntaxError
func (e *ParseError) Code() ErrorCode         { return e.ErrorCode }
func (e *ParseError) GetFile() string         { return e.File }
func (e *ParseError) GetHints() []string      { return e.Hints }
func (e *ParseError) GetDetails() []Highlight { return nil }
func (e *ParseError) Hint(hint string)        { e.Hints = append(e.Hints, hint) }
func (e *ParseError) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// TypeError
func (e *TypeError) GetRange() ranges.Range  { return e.Range }
func (e *TypeError) Code() ErrorCode         { return e.ErrorCode }
func (e *TypeError) GetFile() string         { return e.File }
func (e *TypeError) GetHints() []string      { return e.Hints }
func (e *TypeError) GetDetails() []Highlight { return e.Details }
func (e *TypeError) Hint(hint string)        { e.Hints = append(e.Hints, hint) }
func (e *TypeError) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// Warning
func (e Warning) GetRange() ranges.Range  { return e.Range }
func (e Warning) AtRange() ranges.Range   { return e.Range }
func (e Warning) Code() ErrorCode         { return e.ErrorCode }
func (e Warning) GetFile() string         { return e.File }
func (e Warning) GetHints() []string      { return e.Hints }
func (e Warning) GetDetails() []Highlight { return e.Details }
func (e *Warning) Hint(hint string)       { e.Hints = append(e.Hints, hint) }
func (e *Warning) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// ReferenceError
func (e *ReferenceError) GetFile() string         { return e.File }
func (e *ReferenceError) GetRange() ranges.Range  { return e.Range }
func (e *ReferenceError) AtRange() ranges.Range   { return e.Range }
func (e *ReferenceError) Code() ErrorCode         { return e.ErrorCode }
func (e *ReferenceError) GetHints() []string      { return e.Hints }
func (e *ReferenceError) GetDetails() []Highlight { return e.Details }
