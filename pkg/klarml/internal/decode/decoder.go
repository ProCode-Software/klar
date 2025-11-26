package decode

import (
	"bytes"
	"io"
	"reflect"

	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/context"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

type parseFlags uint8

const (
	topLevel        parseFlags = 1 << iota
	comma                      // Values separated by commas
	structLiteral              // for parseStruct
	continuedString            // String continued from number
	object
)

type decodeFunc func(reflect.Value, *Decoder, parseFlags) (ast.Node, error)

var DecodeCache = MakeCache[reflect.Type, decodeFunc]()

const (
	BufferSize = 64
	MaxDepth   = 10000
)

type Decoder struct {
	Buffer []byte
	Reader io.Reader
	Flags  flags.Flags
	Pos    int // Buffer position
	Depth  int // For nested keys
	Offset int // File position

	Document *ast.Document
	Context  *context.Context
}

func NewBufferDecoder(buf []byte, f ...flags.Flags) *Decoder {
	return &Decoder{
		Document: &ast.Document{},
		Buffer:   buf,
		Flags:    flags.Parse(f...),
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
	}
}

// Looks up a decoder or creates one if it doesn't exist.
func (d *Decoder) lookupDecodeFunc(rt reflect.Type) decodeFunc {
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
	marsh := d.lookupDecodeFunc(rt)
	// Refill if needed. Not needed if the bytes are buffered
	if err := d.Refill(); err != nil && err != EOF {
		return err
	}
	// If this returns an error, the document is empty (treated as nil)
	if err := d.SkipSpaceNewline(); err != nil && err != EOF {
		return err
	}
	body, err := marsh(rv, d, topLevel)
	if err != nil {
		return err
	}
	if body != nil { // TODO: check if all non-nil results are Values
		d.Document.Body = body.(ast.Value)
	}
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

func (d *Decoder) increaseDepth() error {
	if d.Depth++; d.Depth > MaxDepth {
		return errors.ErrMaxDepth
	}
	return nil
}

func (d *Decoder) decreaseDepth() {
	if d.Depth--; d.Depth < 0 {
		panic("d.Depth is negative")
	}
}
