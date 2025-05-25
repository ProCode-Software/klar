package errors

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

type KlarError interface {
	error
	At() lexer.Position
	Code() ErrorCode
}

type ErrorCode int

func (e ParseError) At() lexer.Position { return e.Position }
func (e ParseError) Code() ErrorCode    { return e.Type }
