package decode

import (
	"bytes"
	"io"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type DecodeFunc func(byte, reflect.Value, *Decoder) error

var DecodeCache = MakeCache[reflect.Type, DecodeFunc]()

type Decoder struct {
	Buffer    []byte
	Reader    io.Reader
	Flags     flags.Flags
	Prev, Pos int
}

func NewBufferDecoder(buf []byte, f ...flags.Flags) *Decoder {
	return &Decoder{
		Buffer: buf,
		Flags:  flags.Parse(f...),
	}
}

func NewStreamDecoder(r io.Reader, f ...flags.Flags) *Decoder {
	var buf []byte
	if _, ok := r.(*bytes.Buffer); !ok {
		buf = make([]byte, 0)
	}
	return &Decoder{
		Buffer: buf,
		Reader: r,
		Flags:  flags.Parse(f...),
	}
}

func lookupMarshallFunc(rt reflect.Type) DecodeFunc {
	if marsh, ok := DecodeCache.Get(rt); ok {
		return marsh
	}
	marsh := makeMarshallFunc(rt)
	DecodeCache.Set(rt, marsh)
	return marsh
}

func makeMarshallFunc(rt reflect.Type) DecodeFunc {
	kind := rt.Kind()
	switch kind {
	case reflect.String:

	case reflect.Bool:

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

	case reflect.Float32, reflect.Float64:

	case reflect.Map:

	case reflect.Struct:

	case reflect.Slice:

	case reflect.Array:

	case reflect.Pointer:

	case reflect.Interface:
	default:
		
	}
	return nil
}

func (d *Decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &errors.InvalidUnmarshallError{Type: reflect.TypeOf(v)}
	}
	rv = rv.Elem() // Known pointer
	rt := rv.Type()
	marsh := lookupMarshallFunc(rt)
	marsh(0, reflect.Value{}, nil)
	return nil
}

func (d *Decoder) Curr() byte {
	return d.Buffer[d.Pos]
}

func (d *Decoder) HasBytes() bool {
	return d.Pos < len(d.Buffer)
}

func (d *Decoder) Advance() byte {
	b := d.Buffer[d.Pos]
	d.Pos++
	return b
}
