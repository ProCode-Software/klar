// Package klon implements a parser, encoder, and decoder for Klon, an object notation format used by Klar configurations and manifests.
package klon

import (
	"io"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

// Unmarshall
// =======

// Unmarshall decodes a byte slice into v. v must be a non-nil pointer.
func Unmarshall(b []byte, v any, f ...klonflags.Flags) error {
	return decode(newBufferReader(b), nil, v, f...)
}

// UnmarshallRead decodes from r into v.
func UnmarshallRead(r io.Reader, v any, f ...klonflags.Flags) error {
	return decode(newStreamReader(r), nil, v, f...)
}

// UnmarshallReadContext is [UnmarshallRead], using a [Context] to define classes
// and enums.
func UnmarshallReadContext(r io.Reader, v any, ctx *Context, f ...klonflags.Flags) error {
	return decode(newStreamReader(r), ctx, v, f...)
}

// UnmarshallContext is [Unmarshall], using a [Context] to define classes
// and enums.
func UnmarshallContext(data []byte, v any, ctx *Context, f ...klonflags.Flags) error {
	return decode(newBufferReader(data), ctx, v, f...)
}

// UnmarshallDocument decodes a pre-parsed document into v.
func UnmarshallDocument(d *ast.Document, v any, f ...klonflags.Flags) error {
	return decodeDocument(d, nil, v, f...)
}

// UnmarshallDocumentContext is [UnmarshallDocument], using a
// [Context] to define classes and enums.
func UnmarshallDocumentContext(d *ast.Document, v any, ctx *Context, f ...klonflags.Flags) error {
	return decodeDocument(d, ctx, v, f...)
}

// NewContext creates a new [Context] for custom decoding behavior.
func NewContext() *Context {
	return &Context{}
}

// Parse
// ========

// Parse parses a byte slice into an [ast.Document].
func Parse(b []byte) (*ast.Document, []error) {
	return newBufferReader(b).parseDocument()
}

// ParseReader parses from r into an [ast.Document].
func ParseReader(r io.Reader) (*ast.Document, []error) {
	return newStreamReader(r).parseDocument()
}
