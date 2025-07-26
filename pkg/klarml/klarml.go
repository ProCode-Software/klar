package klarml

import (
	"io"

	"github.com/ProCode-Software/klar/pkg/klarml/internal/decode"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type (
	ExpectedEOFError       = errors.ExpectedEOFError
	ExpectedTokenError     = errors.ExpectedTokenError
	InvalidUnmarshallError = errors.InvalidUnmarshallError
	NumberRangeError       = errors.NumberRangeError
	TypeError              = errors.TypeError
)

var (
	UnexpectedBracketError  = decode.ErrUnexpectedBracket
	UnterminatedArrayError  = decode.ErrUnterminatedArray
	UnterminatedStringError = decode.ErrUnterminatedString
)

func Unmarshall(data []byte, v any, f ...flags.Flags) error {
	d := decode.NewBufferDecoder(data)
	return d.Decode(v)
}

func UnmarshallRead(r io.Reader, v any, f ...flags.Flags) error {
	d := decode.NewStreamDecoder(r)
	return d.Decode(v)
}

func UnmarshallDocument(r any, v any, f ...flags.Flags) error {
	return nil
}
