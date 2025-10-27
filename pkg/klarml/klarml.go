package klarml

import (
	"io"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/context"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/decode"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type (
	ExpectedEOFError       = errors.ExpectedEOF
	ExpectedTokenError     = errors.ExpectedToken
	InvalidUnmarshallError = errors.InvalidUnmarshall
	NumberRangeError       = errors.NumberRange
	TypeError              = errors.TypeError
)

var (
	UnexpectedBracketError  = decode.ErrUnexpectedBracket
	UnterminatedArrayError  = decode.ErrUnterminatedArray
	UnterminatedStringError = decode.ErrUnterminatedString
)

func Unmarshall(b []byte, v any, f ...flags.Flags) error {
	d := decode.NewBufferDecoder(b)
	return d.Decode(v)
}

// UnmarshallContext is [Unmarshall], using a [context.Context] to define classes
// and enums.
func UnmarshallContext(data []byte, v any, ctx *context.Context, f ...flags.Flags) error {
	d := decode.NewBufferDecoder(data)
	d.Context = ctx
	return d.Decode(v)
}

func UnmarshallRead(r io.Reader, v any, f ...flags.Flags) error {
	d := decode.NewStreamDecoder(r)
	return d.Decode(v)
}

// UnmarshallContext is [UnmarshallRead], using a [context.Context] to define classes
// and enums.
func UnmarshallReadContext(r io.Reader, v any, ctx *context.Context, f ...flags.Flags) error {
	d := decode.NewStreamDecoder(r)
	d.Context = ctx
	return d.Decode(v)
}

func UnmarshallDocument(d *ast.Document, v any, f ...flags.Flags) error {
	return nil
}

// UnmarshallDocumentContext is [UnmarshallDocumentContext], using a
// [context.Context] to define classes and enums.
func UnmarshallDocumentContext(d *ast.Document, v any, ctx *context.Context, f ...flags.Flags) error {
	return nil
}

func NewContext() *context.Context {
	return &context.Context{}
}
