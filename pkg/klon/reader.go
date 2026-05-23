package klon
import (
	"io"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

const BufferSize = 64

type bailout struct{}

type reader struct {
	buffer     []byte
	reader     io.Reader
	pos        int // Buffer position
	offset     lexer.Position
	curr       Token
	peek       Token
	hasCurr    bool
	hasPeek    bool
	parseFlags uint8

	depth      int
	lastDashes int
	vars       map[string]ast.Value
	ctx        *Context
	flags      klonflags.Flags
	errs       []error
	comments   []*ast.Comment
	}

func newBufferReader(buf []byte, f ...klonflags.Flags) *reader {
	return &reader{
		buffer: buf,
		offset: lexer.Position{1, 1},
		flags:  parseFlags(f...),
	}
}

func newStreamReader(r io.Reader, f ...klonflags.Flags) *reader {
	return &reader{
		buffer: make([]byte, 0, BufferSize),
		reader: r,
		offset: lexer.Position{1, 1},
		flags:  parseFlags(f...),
	}
}

// Rune reader
// ===============

func (rd *reader) needsMore() bool {
	return rd.pos >= len(rd.buffer)-1
}

// Refill fills the buffer to full capacity if needed. Refill returns an error
// if another byte cannot be read.
func (rd *reader) refill() error {
	if rd.reader == nil {
		if rd.pos >= len(rd.buffer) {
			return io.EOF
		}
		return nil
	}
	if rd.pos < len(rd.buffer) {
		// Copy remaining bytes to the beginning of the buffer
		copy(rd.buffer, rd.buffer[rd.pos:])
		rd.buffer = rd.buffer[:len(rd.buffer)-rd.pos]
		rd.pos = 0
	} else {
		rd.buffer = rd.buffer[:0]
		rd.pos = 0
	}

	n, err := rd.reader.Read(rd.buffer[len(rd.buffer):cap(rd.buffer)])
	if n > 0 {
		rd.buffer = rd.buffer[:len(rd.buffer)+n]
	}
	if err != nil && (n == 0 || err != io.EOF) {
		return err
	}
	if len(rd.buffer) == 0 {
		return io.EOF
	}
	return nil
}

// tryRefill refills the buffer if needed. If the buffer can't be refilled,
// tryRefil returns eof if err == io.EOF, otherwise it panics.
func (rd *reader) tryRefill() (eof error) {
	if rd.needsMore() {
		if err := rd.refill(); err != nil {
			if err == io.EOF {
				return io.EOF
			}
			panic(ReadError{err})
		}
	}
	return nil
}

func handlePanic(e *error) {
	switch err := recover().(type) {
	case nil:
		return
	case bailout:
		return
	case ReadError:
		*e = err
	default:
		panic(err)
	}
}
