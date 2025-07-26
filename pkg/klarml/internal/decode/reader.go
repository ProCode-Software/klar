package decode

import (
	"io"
	"unicode"

	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
)

var EOF = io.EOF

func (d *Decoder) NeedsMore() bool {
	return d.Pos >= len(d.Buffer)-1
}

// Refill fills the buffer to full capacity if needed. Refill returns an error
// if another byte cannot be read.
func (d *Decoder) Refill() error {
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

func (d *Decoder) ReadN(n int) ([]byte, error) {
	if d.Pos+n > len(d.Buffer) {
		if err := d.Refill(); err != nil {
			return nil, err
		}
	}
	return d.Buffer[d.Pos : d.Pos+n], nil
}

func (d *Decoder) Curr() byte {
	return d.Buffer[d.Pos]
}

// Advance returns the current byte and moves to the next byte. Advance will try to
// refill the buffer, returning a nil error if it is safe to call d.Curr().
func (d *Decoder) Advance() (byte, error) {
	curr := d.Curr()
	d.Pos++
	d.FilePos++
	d.Col++
	if curr == '\n' {
		d.Col = 1
		d.Line++
	}
	if d.NeedsMore() {
		if err := d.Refill(); err != nil {
			if err != EOF || d.Pos >= len(d.Buffer) {
				return curr, err
			}
		}
	}
	return curr, nil
}

func (d *Decoder) Expect(exp byte, e ...error) error {
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

func (d *Decoder) SkipSpace() error {
	return d.skipws(false)
}

func (d *Decoder) SkipSpaceNewline() error {
	return d.skipws(true)
}

func (d *Decoder) skipws(includingNewline bool) error {
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
