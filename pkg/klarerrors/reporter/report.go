package reporter

import (
	"bytes"

	"github.com/ProCode-Software/klar/internal/errors"
)

func (r *Reporter) Report(e errors.CompileError) (n int64, err error) {
	buf := &bytes.Buffer{}
	return buf.WriteTo(r.Output)
}
