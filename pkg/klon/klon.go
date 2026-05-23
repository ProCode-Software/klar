package klon

import (
	"io"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

func Unmarshall(b []byte, v any, f ...klonflags.Flags) error {
	return decode(newBufferReader(b), nil, v, f...)
}

func UnmarshallRead(r io.Reader, v any, f ...klonflags.Flags) error {
	return decode(newStreamReader(r), nil, v, f...)
}

// UnmarshallContext is [UnmarshallRead], using a [Context] to define classes
// and enums.
func UnmarshallReadContext(r io.Reader, v any, ctx *Context, f ...klonflags.Flags) error {
	return decode(newStreamReader(r), ctx, v, f...)
}

// UnmarshallContext is [Unmarshall], using a [Context] to define classes
// and enums.
func UnmarshallContext(data []byte, v any, ctx *Context, f ...klonflags.Flags) error {
	return decode(newBufferReader(data), ctx, v, f...)
}

func UnmarshallDocument(d *ast.Document, v any, f ...klonflags.Flags) error {
	return decodeDocument(d, nil, v, f...)
}

// UnmarshallDocumentContext is [UnmarshallDocument], using a
// [Context] to define classes and enums.
func UnmarshallDocumentContext(d *ast.Document, v any, ctx *Context, f ...klonflags.Flags) error {
	return decodeDocument(d, ctx, v, f...)
}

func NewContext() *Context {
	return &Context{}
}

func Parse(b []byte) (*ast.Document, []error) {
	return newBufferReader(b).parseDocument()
}
