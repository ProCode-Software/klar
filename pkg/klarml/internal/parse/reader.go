package parse

import (
	"io"
	"slices"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
)

const (
	BufferSize = 64
	MaxDepth   = 10000
)

type Parser struct {
	Buffer []byte
	Reader io.Reader
	Pos    int // Buffer position
	Depth  int // For nested keys
	Offset int // File position
}

var EOF = io.EOF

func (d *Parser) NeedsMore() bool {
	return d.Pos >= len(d.Buffer)-1
}

func (d *Parser) Overflow() bool {
	return d.Pos >= len(d.Buffer)
}

// Refill fills the buffer to full capacity if needed. Refill returns an error
// if another byte cannot be read.
func (d *Parser) Refill() error {
	if d.Reader == nil {
		if d.NeedsMore() {
			return EOF
		}
		return nil
	}
	if !d.NeedsMore() && d.Buffer[0] != 0 {
		return nil
	}
	prevPos := d.Pos
	d.Pos = 0
	prelen := len(d.Buffer)
	canNext := func(err error) error {
		if err != EOF || d.Pos >= len(d.Buffer)-1 {
			return err
		}
		return nil
	}

	d.Buffer = append(d.Buffer, make([]byte, BufferSize-len(d.Buffer))...)
	n, err := d.Reader.Read(d.Buffer)
	if err != nil && n == 0 {
		d.Buffer = d.Buffer[:prelen]
		d.Pos = prevPos
		return canNext(err)
	}
	d.Buffer = d.Buffer[:n]
	return canNext(EOF)
}

func (d *Parser) ReadN(n int) ([]byte, error) {
	if d.Pos+n > len(d.Buffer) {
		if err := d.Refill(); err != nil {
			return nil, err
		}
	}
	return d.Buffer[d.Pos : d.Pos+n], nil
}

func (d *Parser) Curr() byte {
	return d.Buffer[d.Pos]
}

// Advance returns the current byte and moves to the next byte. Advance will try to
// refill the buffer, returning a nil error if it is safe to call d.Curr().
func (d *Parser) Advance() (byte, error) {
	curr := d.Curr()
	d.Pos++
	d.Offset++
	if d.NeedsMore() {
		if err := d.Refill(); err != nil {
			if err != EOF || d.Pos >= len(d.Buffer) {
				return curr, err
			}
		}
	}
	return curr, nil
}

func (d *Parser) ExpectOne(exp ...byte) error {
	got := d.Curr()
	if !slices.Contains(exp, got) {
		return &errors.ExpectedTokenError{Expected: exp[0], Got: got}
	}
	_, err := d.Advance()
	return err
}

func (d *Parser) Expect(exp byte, e ...error) error {
	got := d.Curr()
	if got != exp {
		if len(e) > 0 {
			return e[0]
		}
		return &errors.ExpectedTokenError{Expected: exp, Got: got}
	}
	_, err := d.Advance()
	return err
}

func (d *Parser) ExpectSpacesThen(exp byte) error {
	switch err := d.SkipSpace(); err {
	case EOF:
		return &errors.UnexpectedEOFError{Expected: exp}
	case nil:
		return d.Expect(exp)
	default:
		return err
	}
}

func (d *Parser) SkipSpace() error {
	return d.skipws(false)
}

func (d *Parser) SkipSpaceNewline() error {
	return d.skipws(true)
}

func (d *Parser) skipws(includingNewline bool) error {
	if d.Pos >= len(d.Buffer) {
		return EOF
	}
	for {
		curr := d.Curr()
		if !unicode.IsSpace(rune(curr)) {
			return nil
		}
		if curr == '\n' && !includingNewline {
			return nil
		}
		if _, err := d.Advance(); err != nil {
			return err
		}
	}
}

func (d *Parser) CurrRune() (r rune, size int) {
	return utf8.DecodeRune(d.Buffer[d.Pos:])
}

func (d *Parser) AdvanceN(n int) error {
	d.Pos += n
	d.Offset += n
	if d.NeedsMore() {
		if err := d.Refill(); err != nil {
			if err != EOF || d.Pos >= len(d.Buffer) {
				return err
			}
		}
	}
	return nil
}
