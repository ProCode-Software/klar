package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type KlarError interface {
	error
	At() lexer.Position
	AtRange() ranges.Range
	Code() ErrorCode
	GetHints() []string
}

//go:generate stringer -type=ErrorCode
type (
	Ranges      = []ranges.Range
	ErrorParams map[string]any
	ErrorCode   int
)

const (
	SyntaxErrorPrefix ErrorCode = iota * 100
	WarningPrefix
	TypeErrorPrefix
	ReferenceErrorPrefix
)

func (e ParseError) At() lexer.Position {
	if ranges.IsZeroPosition(e.Position) {
		return e.AtRange().Start
	}
	return e.Position
}

func (e ParseError) AtRange() ranges.Range {
	if ranges.IsZeroPosition(e.Range.Start) && e.Node != nil {
		return e.Node.Base().Range
	}
	return e.Range
}

// SyntaxError
func (e ParseError) Code() ErrorCode    { return e.ErrorCode }
func (e ParseError) GetHints() []string { return e.Hints }
func (e *ParseError) Hint(hint string)     { e.Hints = append(e.Hints, hint) }

// TypeError
func (e TypeError) At() lexer.Position    { return e.Range.Start }
func (e TypeError) AtRange() ranges.Range { return e.Range }
func (e TypeError) Code() ErrorCode       { return e.ErrorCode }
func (e TypeError) GetHints() []string    { return e.Hints }
func (e *TypeError) Hint(hint string)     { e.Hints = append(e.Hints, hint) }
func (e *TypeError) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}

// Warning
func (e Warning) At() lexer.Position    { return e.Range.Start }
func (e Warning) AtRange() ranges.Range { return e.Range }
func (e Warning) Code() ErrorCode       { return e.ErrorCode }
func (e Warning) GetHints() []string    { return e.Hints }
func (e *Warning) Hint(hint string)     { e.Hints = append(e.Hints, hint) }
func (e *Warning) Hintf(hint string, a ...any) {
	e.Hints = append(e.Hints, fmt.Sprintf(hint, a...))
}
