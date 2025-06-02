package errors

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type KlarError interface {
	error
	At() lexer.Position
	AtRange() ranges.Range
	Code() ErrorCode
}

//go:generate stringer -type=ErrorCode
type ErrorCode int

const (
	ParseErrorPrefix ErrorCode = iota * 100
	TypeErrorPrefix
	ReferenceErrorPrefix
)

func (e ParseError) At() lexer.Position    { return e.Position }
func (e ParseError) AtRange() ranges.Range { return e.Range }
func (e ParseError) Code() ErrorCode       { return e.ErrorCode }

func (e TypeError) At() lexer.Position    { return e.Range.Start }
func (e TypeError) AtRange() ranges.Range { return e.Range }
func (e TypeError) Code() ErrorCode       { return e.ErrorCode }
