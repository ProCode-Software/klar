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
	GetDetails() []ErrorDetail
	GetRanges() []ranges.Range
}

//go:generate stringer -type=ErrorCode
type (
	Ranges      = []ranges.Range
	ErrorParams map[string]any
	ErrorCode   int
	ErrorKind   int
	ErrorDetail struct {
		Range       ranges.Range
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
	if ranges.IsZeroPosition(e.Position) {
		return e.getRange()
	}
	return ranges.Range{e.Position, ranges.Add(e.Position, 0, 1)}
}

func (e ParseError) getRange() ranges.Range {
	if ranges.IsZeroPosition(e.Range.Start) && e.Node != nil {
		return e.Node.GetRange()
	} else if !ranges.IsZeroPosition(e.Token.Position) {
		return ranges.FromToken(e.Token)
	}
	return e.Range
}

// SyntaxError
func (e ParseError) Code() ErrorCode           { return e.ErrorCode }
func (e ParseError) GetFile() string           { return e.File }
func (e ParseError) GetHints() []string        { return e.Hints }
func (e ParseError) GetDetails() []ErrorDetail { return nil }
func (e *ParseError) Hint(hint string)         { e.Hints = append(e.Hints, hint) }
func (e ParseError) GetRanges() []ranges.Range {
	return []ranges.Range{e.getRange()}
}

// TypeError
func (e TypeError) At() ranges.Range          { return e.Range }
func (e TypeError) GetRanges() []ranges.Range { return e.Ranges }
func (e TypeError) Code() ErrorCode           { return e.ErrorCode }
func (e TypeError) GetFile() string           { return e.File }
func (e TypeError) GetHints() []string        { return e.Hints }
func (e TypeError) GetDetails() []ErrorDetail { return e.Details }
func (e *TypeError) Hint(hint string)         { e.Hints = append(e.Hints, hint) }
func (e *TypeError) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// Warning
func (e Warning) At() ranges.Range          { return e.Range }
func (e Warning) AtRange() ranges.Range     { return e.Range }
func (e Warning) Code() ErrorCode           { return e.ErrorCode }
func (e Warning) GetFile() string           { return e.File }
func (e Warning) GetRanges() []ranges.Range { return e.Ranges }
func (e Warning) GetHints() []string        { return e.Hints }
func (e Warning) GetDetails() []ErrorDetail { return e.Details }
func (e *Warning) Hint(hint string)         { e.Hints = append(e.Hints, hint) }
func (e *Warning) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// ReferenceError
func (e ReferenceError) GetFile() string           { return e.File }
func (e ReferenceError) At() ranges.Range          { return e.Range }
func (e ReferenceError) AtRange() ranges.Range     { return e.Range }
func (e ReferenceError) Code() ErrorCode           { return e.ErrorCode }
func (e ReferenceError) GetHints() []string        { return e.Hints }
func (e ReferenceError) GetDetails() []ErrorDetail { return e.Details }
func (e ReferenceError) GetRanges() []ranges.Range { return []ranges.Range{e.Range} }
