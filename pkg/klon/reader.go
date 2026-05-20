package klon

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

const BufferSize = 64

type bailout struct{}

type reader struct {
	buffer     []byte
	reader     io.Reader
	pos        int // Buffer position
	offset     lexer.Position
	curr       *Token
	peek       *Token
	parseFlags uint8

	depth    int
	vars     map[string]ast.Value
	ctx      *Context
	flags    Flags
	errs     []error
	comments []*ast.Comment
}

func newBufferReader(buf []byte, f ...Flags) *reader {
	return &reader{
		buffer: buf,
		offset: lexer.Position{1, 1},
		flags:  parseFlags(f...),
	}
}

func newStreamReader(r io.Reader, f ...Flags) *reader {
	var buf []byte
	if _, ok := r.(*bytes.Buffer); !ok {
		buf = make([]byte, BufferSize)
	}
	return &reader{
		buffer: buf,
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
	if rd.curr != nil {
		return *rd.curr
	}
	tok := rd.readToken()
	rd.curr = &tok
	return tok
}

// peekTok returns the token after the current token without advancing r.
func (rd *reader) peekTok() Token {
	if rd.peek != nil {
		return *rd.peek
	}
	next := rd.readToken()
	rd.peek = &next
	return next
}

// advanceTok returns the current token and advances r.
func (rd *reader) advanceTok() Token {
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
		Range: tokenRange(tok),
		Token: tok,
		Text:  text,
	})
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
	return
}

func (rd *reader) resetParseFlags(old uint8) { rd.parseFlags = old }

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
