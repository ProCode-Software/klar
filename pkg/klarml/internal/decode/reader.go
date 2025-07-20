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

func (d *Decoder) Refill() error {
	if d.Reader == nil {
		if d.NeedsMore() {
			return EOF
		}
		return nil
	}
	var start int
	if d.Pos >= len(d.Buffer)-1 {
		start = 0
	} else {
		remaining := len(d.Buffer) - d.Pos
		copy(d.Buffer[:remaining], d.Buffer[d.Pos:])
		start = remaining
	}
	if d.Buffer == nil {
		d.Buffer = make([]byte, 64)
	}
	n, err := d.Reader.Read(d.Buffer[start:])
	if err != nil && start+n == 0 {
		return err
	}
	d.Buffer = d.Buffer[:start+n]
	// Reset pos
	d.Pos = 0
	return nil
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

func (d *Decoder) Advance() (byte, error) {
	curr := d.Curr()
	d.Pos++
	if d.NeedsMore() {
		if err := d.Refill(); err != nil {
			return 0, err
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
	for {
		curr := d.Curr()
		if curr != '\n' && !unicode.IsSpace(rune(curr)) {
			return nil
		}
		if _, err := d.Advance(); err != nil {
			return err
		}
	}
}
func (d *Decoder) SkipSpaceNewline() error {
	for {
		curr := d.Curr()
		if !unicode.IsSpace(rune(curr)) {
			return nil
		}
		if _, err := d.Advance(); err != nil {
			return err
		}
	}
}
