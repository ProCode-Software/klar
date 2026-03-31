package reporter

import (
	"fmt"
	"strconv"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (r *Reporter) checkForFile(name string) *file {
	file, ok := r.files[name]
	if !ok {
		panic("file not loaded into Reporter: " + name)
	}
	return file
}

// getTokenIndexForLine returns the index of the first token on line.
// getTokenIndexForLine may return a token that begins on an earlier line,
// but ends on line, such as multiline strings.
func (f *file) getTokenIndexForLine(line uint32) int {
	// Use cached file.lastLine and lastLineTok
	var currTok int
	if f.lastLine > 0 && f.lastLine <= line {
		currTok = f.lastLineTok
	}
	for i := currTok; i < len(f.tokens); i++ {
		// Check end line in case of multiline strings that end here
		if ranges.TokenEnd(f.tokens[i]).Line >= line {
			currTok = i
			break
		}
	}
	// Update cache
	f.lastLine = line
	f.lastLineTok = currTok
	return currTok
}

func (r *Reporter) appendRune(c rune, color string) {
	r.buf.WriteString(ansi.Color(color, string(c)))
}

func (r *Reporter) appendString(s string, color string) {
	r.buf.WriteString(ansi.Color(color, s))
}

func (r *Reporter) appendNumber(n uint32, color string) {
	r.buf.WriteString(ansi.Color(color, strconv.FormatUint(uint64(n), 10)))
}

func (r *Reporter) appendf(color, format string, a ...any) {
	fmt.Fprintf(r.buf, ansi.Color(color, format), a...)
}

func (r *Reporter) appendSpace(n int) { r.buf.Write(char.Repeat(' ', n)) }
func (r *Reporter) newline()          { r.buf.WriteByte('\n') }
func (r *Reporter) blankLine()        { r.buf.Write([]byte("\n\n")) }

func (r *Reporter) padding(lastCol, currCol uint32) {
	if padding := int(currCol) - int(lastCol); padding > 0 {
		r.appendSpace(padding)
	} else if padding < 0 {
		panic("negative column offset")
	}
}
