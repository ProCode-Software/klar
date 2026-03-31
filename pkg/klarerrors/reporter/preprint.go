package reporter

import (
	"cmp"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"golang.org/x/term"
)

// init initializes the reporter.
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
	if r.ColorPalette == nil {
		r.ColorPalette = DefaultColorPalette()
	}
	getTermWidth(r.Output)
}

// getTermWidth gets the width of the terminal. If it fails, it sets termWidth to 0.
func getTermWidth(w io.Writer) {
	if w, ok := w.(*os.File); ok {
		var err error
		termWidth, _, err = term.GetSize(int(w.Fd()))
		if err != nil {
			termWidth = 0
		}
	}
}

// printDivider prints a divider line that is the width of the terminal,
// followed by a newline.
func (r *Reporter) printDivider() {
	div := char.RepeatRune(r.CharacterSet.ErrorDivider, max(3, termWidth))
	r.buf.WriteString(ansi.Partial(r.ColorPalette.Divider))
	r.buf.Write(div)
	r.buf.WriteString(ansi.Reset())
	r.buf.WriteByte('\n')
}

// digitLen returns the number of digits required to print x.
func digitLen(x uint32) int {
	if x < 10 {
		return 1
	} else if x < 100 {
		return 2
	} else if x < 1000 {
		return 3
	}
	return len(strconv.FormatUint(uint64(x), 10))
}

// sortHighlights sorts highlights by the order in which they will be printed.
// Highlights on the earliest line are printed first. If two highlights are on
// the same line, the leftmost highlight is printed first.
func sortHighlights(highlights []errors.Highlight) {
	slices.SortFunc(highlights, func(a, b errors.Highlight) int {
		return cmp.Or(
			cmp.Compare(a.Range.Start.Line, b.Range.Start.Line),
			cmp.Compare(a.Range.End.Col, b.Range.End.Col),
		)
	})
}
