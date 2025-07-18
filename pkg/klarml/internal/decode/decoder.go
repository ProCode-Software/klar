package decode

import (
	"bytes"
	"io"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type unmarshaller func(byte, reflect.Value, *Decoder) error

var DecodeCache = MakeCache[reflect.Type, unmarshaller]()

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
		buf = make([]byte, 0, 64)
	}
	return &Decoder{
		Buffer: buf,
		Reader: r,
		Flags:  flags.Parse(f...),
	}
}

func lookupMarshallFunc(rt reflect.Type) unmarshaller {
	if marsh, ok := DecodeCache.Get(rt); ok {
		return marsh
	}
	marsh := makeDefaultMarshaller(rt)
	DecodeCache.Set(rt, marsh)
	return marsh
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

func (d *Decoder) Read() error {
	if d.Reader == nil {
		return nil
	}
	_, err := d.Reader.Read(d.Buffer)
	if err != nil {
		return err
	}
	return nil
}