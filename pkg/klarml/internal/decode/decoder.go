package decode

import (
	"bytes"
	goerrors "errors"
	"io"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type decodeFunc func(reflect.Value, *Decoder) (ast.Node, error)

var (
	DecodeCache    = MakeCache[reflect.Type, decodeFunc]()
	ErrExpectedEOF = goerrors.New("expected end of file")
)

const BufferSize = 64

type Decoder struct {
	Buffer       []byte
	Reader       io.Reader
	Flags        flags.Flags
	PrevEnd, Pos int
	FilePos      int
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
		buf = make([]byte, BufferSize)
	}
	return &Decoder{
		Buffer: buf,
		Reader: r,
		Flags:  flags.Parse(f...),
	}
}

func lookupMarshallFunc(rt reflect.Type) decodeFunc {
	if marsh, ok := DecodeCache.Get(rt); ok {
		return marsh
	}
	marsh := makeDefaultDecoder(rt)
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
	if err := d.Refill(); err != nil {
		return err
	}
	if err := d.SkipSpaceNewline(); err != nil {
		return err
	}
	if _, err := marsh(rv, d); err != nil && err != EOF {
		return err
	}
	// Make sure there is nothing else after decoding
	if err := d.SkipSpaceNewline(); err != io.EOF {
		return &errors.ExpectedEOFError{Got: d.Curr()}
	}
	return nil
}

func (d *Decoder) TypeError(rv reflect.Value, got ast.Node) error {
	return &errors.TypeError{
		Expected: rv.Type(),
		Value:    got,
	}
}
