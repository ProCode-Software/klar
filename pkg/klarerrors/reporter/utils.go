package reporter

import (
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

// getTokenIndexForLine finds the first token index mapped to a target line.
func (f *file) getTokenIndexForLine(line uint32) int {
	// Use cached file.lastLine and lastLineTok
	var currTok int
	if f.lastLine > 0 && uint32(f.lastLine) <= line {
		currTok = f.lastLineTok
	}
	for i := currTok; i < len(f.tokens); i++ {
		// Check End.Line in case of multiline strings that end here
		if ranges.FromToken(f.tokens[i]).End.Line >= line {
			currTok = i
			break
		}
	}
	// Update cache
	f.lastLine = int(line)
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
func (r *Reporter) appendColor(color string) {
	r.buf.WriteString(ansi.Partial(color))
}

func (r *Reporter) appendSpace(n int) { r.buf.Write(char.Repeat(' ', n)) }
func (r *Reporter) newline()          { r.buf.WriteByte('\n') }
func (r *Reporter) blankLine()        { r.buf.Write([]byte("\n\n")) }
