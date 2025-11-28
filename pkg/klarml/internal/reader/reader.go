package reader

import (
	"bytes"
	"io"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/klarml/ast"
	"github.com/ProCode-Software/klar/pkg/klarml/context"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/flags"
)

const BufferSize = 64

type ReadError struct {
	Error error
}

type Reader struct {
	buffer []byte
	reader io.Reader
	pos    int // Buffer position
	offset lexer.Position
	curr   *Token
	peek   *Token
	comma  bool

	depth int
	vars  map[string]ast.Value
	ctx   *context.Context
	flags flags.Flags
	errs  []error
}

func NewBufferReader(buf []byte, f ...flags.Flags) *Reader {
	return &Reader{
		buffer: buf,
		flags:  flags.Parse(f...),
	}
}

func NewStreamReader(r io.Reader, f ...flags.Flags) *Reader {
	var buf []byte
	if _, ok := r.(*bytes.Buffer); !ok {
		buf = make([]byte, BufferSize)
	}
	return &Reader{
		buffer: buf,
		reader: r,
		flags:  flags.Parse(f...),
	}
}

// Rune reader
// ===============

func (rd *Reader) needsMore() bool {
	return rd.pos >= len(rd.buffer)-1
}

// Refill fills the buffer to full capacity if needed. Refill returns an error
// if another byte cannot be read.
func (rd *Reader) refill() error {
	if rd.reader == nil {
		if rd.needsMore() {
			return io.EOF
		}
		return nil
	}
	if !rd.needsMore() && rd.buffer[0] != 0 {
		return nil
	}
	prevPos := rd.pos
	rd.pos = 0
	prelen := len(rd.buffer)
	canNext := func(err error) error {
		if err != io.EOF || rd.pos >= len(rd.buffer)-1 {
			return err
		}
		return nil
	}

	rd.buffer = append(rd.buffer, make([]byte, BufferSize-len(rd.buffer))...)
	n, err := rd.reader.Read(rd.buffer)
	if err != nil && n == 0 {
		rd.buffer = rd.buffer[:prelen]
		rd.pos = prevPos
		return canNext(err)
	}
	rd.buffer = rd.buffer[:n]
	return canNext(io.EOF)
}

// tryRefill refills the buffer if needed. If the buffer can't be refilled,
// tryRefil returns eof if err == io.EOF, otherwise it panics.
func (rd *Reader) tryRefill() (eof error) {
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

// Token reader
// ===============

// currTok returns the current token.
func (rd *Reader) currTok() Token {
	if rd.curr != nil {
		return *rd.curr
	}
	tok := rd.readToken()
	rd.curr = &tok
	return tok
}

// peekTok returns the token after the current token without advancing r.
func (rd *Reader) peekTok() Token {
	if rd.peek != nil {
		return *rd.peek
	}
	_ = rd.currTok()
	next := rd.readToken()
	rd.peek = &next
	return next
}

// advanceTok returns the current token and advances r.
func (rd *Reader) advanceTok() Token {
	if rd.peek != nil {
		t := *rd.curr
		rd.curr = rd.peek
		rd.peek = nil
		return t
	}
	curr := rd.currTok()
	new := rd.readToken()
	rd.curr = &new
	rd.peek = nil
	return curr
}
