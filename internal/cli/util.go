package cli

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/char"
)

func FormatDuration(dur time.Duration) string {
	switch {
	case dur >= time.Hour:
		hours := float64(dur) / float64(time.Hour)
		return formatFloat(hours) + "hr"
	case dur >= time.Minute:
		minutes := float64(dur) / float64(time.Minute)
		return formatFloat(minutes) + "m"
	case dur >= time.Second:
		seconds := float64(dur) / float64(time.Second)
		return formatFloat(seconds) + "s"
	case dur >= time.Millisecond:
		ms := float64(dur) / float64(time.Millisecond)
		return formatFloat(ms) + "ms"
	case dur >= time.Microsecond:
		us := float64(dur) / float64(time.Microsecond)
		return formatFloat(us) + "µs"
	default:
		return strconv.FormatInt(int64(dur), 10) + "ns"
	}
}

func formatFloat(f float64) string {
	prec := 2
	if f >= 100 {
		prec--
	}
	s := strconv.FormatFloat(f, 'f', prec, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

type AllWriter interface {
	io.Writer
	io.StringWriter
	io.ByteWriter
}

// Wrap wraps a string to a given width, with an optional left margin
// and first row width. The first row is unindented.
func Wrap(s string, w AllWriter, width, firstRowWidth, margin int) {
	if width <= 0 {
		return
	}
	var (
		textWidth    = max(width-margin, 1) // Width of the text after the margin
		i            int
		needsNewline bool
		currWidth = width
	)
	// Adjust width for the first line
	if firstRowWidth > 0 {
		currWidth = firstRowWidth
	}
	// Writes a newline and the next line's margin
	writeNewline := func() {
		if needsNewline {
			w.WriteByte('\n')
			if margin > 0 {
				w.Write(char.Repeat(' ', margin))
			}
		}
	}

	for i < len(s) {
		end := min(i+currWidth, len(s))

		// Check for an existing newline before breaking
		if newlineI := strings.IndexByte(s[i:end], '\n'); newlineI >= 0 {
			// Don't add a wrap newline if we immediately hit an explicit newline
			if newlineI == 0 && needsNewline {
				needsNewline = false
			}
			writeNewline()
			// Everything up to (including) the newline
			w.WriteString(s[i : i+newlineI+1])
			i += newlineI + 1
			// Write margin for the next line
			if margin > 0 && i < len(s) {
				w.Write(char.Repeat(' ', margin))
			}
			needsNewline = false

			// Switch to the text width for the next lines
			currWidth = textWidth
			continue
		}

		// Calculate the length of the line
		lineLen := end - i
		if end < len(s) {
			if spaceI := strings.LastIndexByte(s[i:end], ' '); spaceI > 0 {
				// Break at the last space
				lineLen = spaceI
			} else {
				// No space found within the width. Find the next space to avoid cutting the word.
				nextBreak := strings.IndexAny(s[end:], " \n")
				if nextBreak >= 0 {
					lineLen = (end - i) + nextBreak
				} else {
					lineLen = len(s) - i // No space found, take the rest of the string
				}
			}
		}
		writeNewline()
		w.WriteString(s[i : i+lineLen])
		i += lineLen

		needsNewline = true
		currWidth = textWidth // Switch to the text width for the next lines

		// Skip spaces after a break
		for i < len(s) && s[i] == ' ' {
			i++
		}
	}
}
