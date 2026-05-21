package klon

import (
	"fmt"
	"io"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
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
	flags      Flags
	errs       []error
	comments   []*ast.Comment
}

func newBufferReader(buf []byte, f ...Flags) *reader {
	return &reader{
		buffer: buf,
		offset: lexer.Position{1, 1},
		flags:  parseFlags(f...),
	}
}

func newStreamReader(r io.Reader, f ...Flags) *reader {
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

// Token reader
// ===============

func (rd *reader) hasTokens() bool {
	return rd.currTok().Kind != EOF
}

// currTok returns the current token.
func (rd *reader) currTok() Token {
	if rd.hasCurr {
		return rd.curr
	}
	rd.curr = rd.readToken()
	rd.hasCurr = true
	return rd.curr
}

// peekTok returns the token after the current token without advancing r.
func (rd *reader) peekTok() Token {
	if rd.hasPeek {
		return rd.peek
	}
	rd.peek = rd.readToken()
	rd.hasPeek = true
	return rd.peek
}

// advanceTok returns the current token and advances r.
func (rd *reader) advanceTok() Token {
	if rd.hasPeek {
		t := rd.curr
		rd.curr = rd.peek
		rd.hasCurr = true
		rd.hasPeek = false
		return t
	}
	t := rd.currTok()
	rd.hasCurr = false
	return t
}

func (rd *reader) skipLines() {
	for rd.currTok().Kind == Newline {
		rd.advanceTok()
	}
}

func (rd *reader) tokenError(code Code, tok Token, msg string, v ...any) {
	var text string
	if len(v) == 0 {
		text = msg
	} else {
		text = fmt.Sprintf(msg, v...)
	}
	rd.errs = append(rd.errs, &Error{
		Code:  code,
		Range: tok.Range(),
		Token: tok,
		Text:  text,
	})
}

func (rd *reader) rangeError(code Code, r ranges.Range, msg string, v ...any) {
	var text string
	if len(v) == 0 {
		text = msg
	} else {
		text = fmt.Sprintf(msg, v...)
	}
	rd.errs = append(rd.errs, &Error{Code: code, Range: r, Text: text})
}

func (rd *reader) expectError(
	exp TokenType, code Code, msg string, v ...any,
) Token {
	if curr := rd.currTok(); curr.Kind != exp {
		rd.tokenError(code, curr, msg, v...)
		return curr
	}
	return rd.advanceTok()
}

func (rd *reader) depthUp() {
	if rd.depth++; rd.depth > MaxDepth {
		rd.tokenError(ErrMaxDepth, rd.currTok(), "Too much nesting")
		panic(bailout{})
	}
}

func (rd *reader) depthDown() {
	if rd.depth--; rd.depth < 0 {
		panic("negative depth")
	}
}

func (rd *reader) addParseFlags(flags uint8) (old uint8) {
	old = rd.parseFlags
	rd.parseFlags |= flags
	if old != rd.parseFlags {
		rd.invalidateCache()
	}
	return
}

func (rd *reader) resetParseFlags(old uint8) {
	if rd.parseFlags != old {
		rd.parseFlags = old
		rd.invalidateCache()
	}
}

func (rd *reader) invalidateCache() {
	if !rd.hasCurr {
		return
	}
	// Back up to the start of the current token
	rd.pos = rd.curr.BufPos
	rd.offset = rd.curr.Pos
	rd.hasCurr = false
	rd.hasPeek = false
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
