package klarml

import (
	"io"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/context"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/decode"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/reader"
)

type (
	ParseError = reader.ParseError
)

func Unmarshall(b []byte, v any, f ...flags.Flags) error {
	return decode.Decode(reader.NewBufferReader(b), nil, v, f...)
}

func UnmarshallRead(r io.Reader, v any, f ...flags.Flags) error {
	return decode.Decode(reader.NewStreamReader(r), nil, v, f...)
}

// UnmarshallContext is [UnmarshallRead], using a [context.Context] to define classes
// and enums.
func UnmarshallReadContext(r io.Reader, v any, ctx *context.Context, f ...flags.Flags) error {
	return decode.Decode(reader.NewStreamReader(r), ctx, v, f...)
}

// UnmarshallContext is [Unmarshall], using a [context.Context] to define classes
// and enums.
func UnmarshallContext(data []byte, v any, ctx *context.Context, f ...flags.Flags) error {
	return decode.Decode(reader.NewBufferReader(data), ctx, v, f...)
}

func UnmarshallDocument(d *ast.Document, v any, f ...flags.Flags) error {
	return decode.DecodeDocument(d, nil, v, f...)
}

// UnmarshallDocumentContext is [UnmarshallDocument], using a
// [context.Context] to define classes and enums.
func UnmarshallDocumentContext(d *ast.Document, v any, ctx *context.Context, f ...flags.Flags) error {
	return decode.DecodeDocument(d, ctx, v, f...)
}

func NewContext() *context.Context {
	return &context.Context{}
}
