package klon

import (
	"encoding"
	"fmt"
	"io"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

var (
	marshallerType    = reflect.TypeFor[Marshaller]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

type encodeFunc = func(reflect.Value, *encoder) error

var encodeCache = makeCache[reflect.Type, encodeFunc]()

type encoder struct {
	writer      io.Writer
	buf         []byte
	flags       klonflags.Flags
	indentSize  int // Number of spaces to indent with
	indentLevel int // Current level while encoding
}

func newBufferEncoder(buf []byte, indent int, f ...klonflags.Flags) *encoder {
	return &encoder{
		buf:        buf,
		flags:      parseFlags(f...),
		indentSize: indent,
	}
}

func newStreamEncoder(w io.Writer, indent int, f ...klonflags.Flags) *encoder {
	return &encoder{
		writer:     w,
		flags:      parseFlags(f...),
		indentSize: indent,
	}
}

func (e *encoder) encode(v any) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() == reflect.Pointer && rv.IsNil()) {
		e.write("none")
		return nil
	}
	encode := e.getEncoder(rv.Type())
	return encode(rv, e)
}

func (e *encoder) getEncoder(rt reflect.Type) (encode encodeFunc) {
	if encode, ok := encodeCache.get(rt); ok {
		return encode
	}

	// Use [Marshaller] or [encoding.TextMarshaler] if the type implements it.
	// Marshaller has priority over TextMarshaler.
	ptr := reflect.PointerTo(rt)
	switch {
	case rt.Implements(marshallerType), ptr.Implements(marshallerType):
		encode = encodeMarshaller
	case rt.Implements(textMarshalerType), ptr.Implements(textMarshalerType):
		encode = encodeTextMarshaler
	default:
		encode = e.makeDefaultEncoder(rt)
	}

	encodeCache.set(rt, encode)
	return encode
}

func (e *encoder) write(s string) error {
	if e.buf != nil {
		e.buf = append(e.buf, s...)
		return nil
	}
	_, err := e.writer.Write([]byte(s))
	return err
}

func (e *encoder) writeSlice(b []byte) error {
	if e.buf != nil {
		e.buf = append(e.buf, b...)
		return nil
	}
	_, err := e.writer.Write(b)
	return err
}

func (e *encoder) writeByte(c byte) error {
	if e.buf != nil {
		e.buf = append(e.buf, c)
		return nil
	}
	_, err := e.writer.Write([]byte{c})
	return err
}

func (e *encoder) writef(f string, v ...any) error {
	if e.buf != nil {
		e.buf = fmt.Appendf(e.buf, f, v...)
		return nil
	}
	_, err := fmt.Fprintf(e.writer, f, v...)
	return err
}
