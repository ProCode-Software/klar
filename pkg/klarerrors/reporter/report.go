package reporter

import (
	"bytes"
	"os"

	"github.com/ProCode-Software/klar/internal/errors"
)

func (r *Reporter) Report(e errors.CompileError) (n int64, err error) {
	r.buf = &bytes.Buffer{}

	return r.buf.WriteTo(r.Output)
}

func (r *Reporter) init() {
	if r.Output == nil {
		r.Output = os.Stderr
	}
	if r.MaxLines <= 0 {
		r.MaxLines = 3
	}
	if r.CharacterSet == nil {
		r.CharacterSet = DefaultCharacterSet()
	}
}

func (r *Reporter) printHeader(e errors.CompileError) {
}
