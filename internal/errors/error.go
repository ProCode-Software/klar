package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
)

type KlarError interface {
	error
	At() ranges.Range
	Code() ErrorCode
	GetHints() []string
	GetFile() string
	GetDetails() []Detail
}

//go:generate stringer -type=ErrorCode
type (
	Ranges      = []ranges.Range
	ErrorParams map[string]any
	ErrorCode   int
	ErrorKind   int
	Detail      struct {
		Range       ranges.Range
		Type        string
		Description string
	}
)

const (
	SyntaxErrorPrefix ErrorCode = iota * 100
	WarningPrefix
	TypeErrorPrefix
	ReferenceErrorPrefix
)

func (e ParseError) At() ranges.Range {
	return e.getRange()
}

func (e ParseError) getRange() ranges.Range {
	if e.Range.Start.Line > 0 {
		return e.Range
	} else if e.Node != nil {
		return e.Node.GetRange()
	} else if e.Token.Position.Line > 0 {
		return ranges.FromToken(e.Token)
	}
	return ranges.Range{e.Position, ranges.Add(e.Position, 0, 1)}
}

// SyntaxError
func (e ParseError) Code() ErrorCode      { return e.ErrorCode }
func (e ParseError) GetFile() string      { return e.File }
func (e ParseError) GetHints() []string   { return e.Hints }
func (e ParseError) GetDetails() []Detail { return nil }
func (e *ParseError) Hint(hint string)    { e.Hints = append(e.Hints, hint) }

// TypeError
func (e TypeError) At() ranges.Range     { return e.Range }
func (e TypeError) Code() ErrorCode      { return e.ErrorCode }
func (e TypeError) GetFile() string      { return e.File }
func (e TypeError) GetHints() []string   { return e.Hints }
func (e TypeError) GetDetails() []Detail { return e.Details }
func (e *TypeError) Hint(hint string)    { e.Hints = append(e.Hints, hint) }
func (e *TypeError) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// Warning
func (e Warning) At() ranges.Range      { return e.Range }
func (e Warning) AtRange() ranges.Range { return e.Range }
func (e Warning) Code() ErrorCode       { return e.ErrorCode }
func (e Warning) GetFile() string       { return e.File }
func (e Warning) GetHints() []string    { return e.Hints }
func (e Warning) GetDetails() []Detail  { return e.Details }
func (e *Warning) Hint(hint string)     { e.Hints = append(e.Hints, hint) }
func (e *Warning) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// ReferenceError
func (e ReferenceError) GetFile() string       { return e.File }
func (e ReferenceError) At() ranges.Range      { return e.Range }
func (e ReferenceError) AtRange() ranges.Range { return e.Range }
func (e ReferenceError) Code() ErrorCode       { return e.ErrorCode }
func (e ReferenceError) GetHints() []string    { return e.Hints }
func (e ReferenceError) GetDetails() []Detail  { return e.Details }
