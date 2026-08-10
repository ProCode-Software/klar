// Package klon implements a parser, encoder, and decoder for Klon, an object
// notation format used by Klar configurations and manifests.
package klon

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

// Unmarshall
// =======

// Unmarshaller is the interface implemented by types that can unmarshall
// themselves from a Klon AST node.
type Unmarshaller interface {
	// UnmarshallKlon (with 2 l's) should be implemented by a pointer receiver.
	UnmarshallKlon(val ast.Value) error
}

// Unmarshall decodes a byte slice into v. v must be a non-nil pointer.
func Unmarshall(b []byte, v any, f ...klonflags.Flags) error {
	return decode(newBufferReader(b), nil, v, f...)
}

// UnmarshallRead decodes from r into v.
func UnmarshallRead(r io.Reader, v any, f ...klonflags.Flags) error {
	return decode(newStreamReader(r), nil, v, f...)
}

// UnmarshallDocument decodes a pre-parsed document into v.
func UnmarshallDocument(d *ast.Document, v any, f ...klonflags.Flags) error {
	return decodeDocument(d, nil, v, f...)
}

// Unmarshall with Context
// ========

// Unmarshall is [Unmarshall], using ctx to define classes and enums.
func (ctx *Context) Unmarshall(data []byte, v any, f ...klonflags.Flags) error {
	return decode(newBufferReader(data), ctx, v, f...)
}

// UnmarshallRead is [UnmarshallRead], using ctx to define classes and enums.
func (ctx *Context) UnmarshallRead(r io.Reader, v any, f ...klonflags.Flags) error {
	return decode(newStreamReader(r), ctx, v, f...)
}

// UnmarshallDocument is [UnmarshallDocument], using ctx to define classes and enums.
func (ctx *Context) UnmarshallDocument(d *ast.Document, v any, f ...klonflags.Flags) error {
	return decodeDocument(d, ctx, v, f...)
}

// Parse
// ========

// Parse parses a byte slice into an [ast.Document].
func Parse(b []byte, f ...klonflags.Flags) (*ast.Document, []*Error) {
	return newBufferReader(b, f...).parseDocument()
}

// ParseRead parses from r into an [ast.Document].
func ParseRead(r io.Reader, f ...klonflags.Flags) (*ast.Document, []*Error) {
	return newStreamReader(r, f...).parseDocument()
}

// Marshall
// =========

type Marshaller interface {
	// The result will be treated as valid Klon. If you want to return a,
	// string, it must be quoted. [Quote] can be used to quote a string
	// into valid Klon.
	MarshallKlon() ([]byte, error)
}

// TODO: In the future, we could maybe add an interface that returns an AST node.
// The issue with that is formatting.

// Marshall serializes a Go value to Klon. The output doesn't end with
// a newline unless the [klonflags.InsertFinalNewline] flag is provided.
// If indent <= 0, objects and lists will be marshalled inline (using brackets)
func Marshall(v any, indent int, f ...klonflags.Flags) ([]byte, error) {
	e := newBufferEncoder(make([]byte, 0, 1024), indent, f...)
	err := e.encode(v)
	return e.buf, err
}

// MarshallWrite serializes a Go value to Klon, writing the output to w.
// The output doesn't end with a newline unless the [klonflags.InsertFinalNewline]
// flag is provided.
// If indent <= 0, objects and lists will be marshalled inline (using brackets)
func MarshallWrite(v any, indent int, w io.Writer, f ...klonflags.Flags) error {
	return newStreamEncoder(w, indent, f...).encode(v)
}

// Quote adds quotes around the provided byte slice, resulting in a valid
// Klon string. If the input can be left unquoted, it will be returned as-is.
// Escape sequences are added as needed. Quote uses single quotes by default,
// but will use double quotes if the input contains single quotes.
func Quote(b []byte) []byte {
	if len(b) == 0 {
		return []byte("''") // Empty strings must be quoted
	}
	orig := b
	// Klon strings must be quoted if any of the following are true:
	// - It contains special punctuation defined in [reader.isPunct]
	// - It is the string "none"
	// - It begins or ends in whitespace
	// - It begins with a quote
	//
	// Strings need to be escaped if they contain non-printable
	// characters (such as control characters) or invalid UTF-8.
	//
	// Quote prefers quoting strings over escaping when possible.
	//
	// TODO: Should valid numbers be quoted? If a number is left unquoted,
	// determinism is lost when unmarshalling into 'any'
	var (
		// All other space characters must be escaped
		canUnquote = len(bytes.Trim(b, " ")) == len(b) &&
			string(b) != "none" &&
			b[0] != '"' && b[0] != '\''
		needEscape     bool
		hasSingleQuote bool
	)
	for canUnquote && len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		b = b[size:]
		switch r {
		case utf8.RuneError, '\\', '$':
			needEscape = true
		case '\n', '@', '[', ']', '{', '}', ':', ',', '.', '"':
			canUnquote = false
		case '\'':
			hasSingleQuote = true
		case '\f', '\r', '\t', '\v', '\x1b':
			// Characters that can be escaped (the string can still be unquoted)
			// See [getEscape]
			needEscape = true
		default:
			if !unicode.IsPrint(r) {
				canUnquote = false
			}
		}
	}
	if canUnquote && !needEscape {
		return orig // No characters need escaping or quoting!
	}

	// Quote and/or escape the string
	res := make([]byte, 0, len(b)+2)
	var quoteStyle byte = '\''
	if !canUnquote {
		if hasSingleQuote {
			quoteStyle = '"'
		}
		res = append(res, quoteStyle)
	}
	for i := 0; i < len(orig); i++ {
		b := orig[i]
		r, size := utf8.DecodeRune(orig[i:])
		switch {
		case b == quoteStyle, b == '$', b == '\\':
			// Need escaping
			// Put a backslash before the character. No letter needed
			res = append(res, '\\', b)
		case r == utf8.RuneError:
			// Read invalid UTF-8 byte-by-byte so each surrogate pair is escaped
			for range size {
				res = fmt.Appendf(res, `\u{%.2x}`, orig[i])
				i++
			}
			i--
			continue
		case unicode.IsPrint(r):
			res = append(res, orig[i:i+size]...) // No escaping needed
		default:
			// Escape nonprintable characters. Code inside \u{...} must
			// be at least 2 digits long
			res = fmt.Appendf(res, `\u{%.2x}`, r)

		// Letters needed. See [getEscape]
		case b == '\f':
			res = append(res, `\f`...)
		case b == '\n':
			res = append(res, `\n`...) // TODO: Does \n need escaping?
		case b == '\r':
			res = append(res, `\r`...)
		case b == '\t':
			res = append(res, `\t`...)
		case b == '\v':
			res = append(res, `\v`...)
		case b == '\x1b':
			res = append(res, `\e`...)
		}
		i += size - 1
	}
	if !canUnquote {
		res = append(res, quoteStyle)
	}
	return res
}
