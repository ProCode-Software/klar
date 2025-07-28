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
	DecodeCache      = MakeCache[reflect.Type, decodeFunc]()
	ErrDocumentEmpty = goerrors.New("document is empty")
)

const BufferSize = 64

type Decoder struct {
	Buffer []byte
	Reader io.Reader
	Flags  flags.Flags
	Pos    int

	Line, Col int
	FilePos   int

	Depth    int // For nested keys
	TopLevel bool

	Document *ast.Document
}

func NewBufferDecoder(buf []byte, f ...flags.Flags) *Decoder {
	return &Decoder{
		Document: &ast.Document{},
		Buffer:   buf,
		Flags:    flags.Parse(f...),
		Line:     1, Col: 1,
	}
}

func NewStreamDecoder(r io.Reader, f ...flags.Flags) *Decoder {
	var buf []byte
	if _, ok := r.(*bytes.Buffer); !ok {
		buf = make([]byte, BufferSize)
	}
	return &Decoder{
		Document: &ast.Document{},
		Buffer:   buf,
		Reader:   r,
		Flags:    flags.Parse(f...),
		Line:     1, Col: 1,
	}
}

// Looks up a decoder or creates one if it doesn't exist.
func (d *Decoder) lookupMarshallFunc(rt reflect.Type) decodeFunc {
	if marsh, ok := DecodeCache.Get(rt); ok {
		return marsh
	}
	marsh := d.makeDefaultDecoder(rt)
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
	marsh := d.lookupMarshallFunc(rt)
	// Refill if needed. Not needed if the bytes are buffered
	if err := d.Refill(); err != nil && err != EOF {
		return err
	}
	// If this returns an error, the document is empty
	if err := d.SkipSpaceNewline(); err != nil {
		if err == EOF {
			return ErrDocumentEmpty
		}
		return err
	}
	body, err := marsh(rv, d)
	if err != nil {
		return err
	}
	d.Document.Body = body.(ast.Value)
	// Make sure there is nothing else after decoding
	switch err := d.SkipSpaceNewline(); err {
	case EOF:
		return nil
	case nil:
		return &errors.ExpectedEOFError{Got: d.Curr()}
	default:
		return err
	}
}

func (d *Decoder) TypeError(rv reflect.Value, got ast.Node) error {
	return &errors.TypeError{
		Expected: rv.Type(),
		Value:    got,
	}
}
