package cli

import (
	"strconv"
	"strings"
	"time"
	"io"

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

func Wrap(s string, w AllWriter, width, margin int) {
	if width <= 0 {
		return
	}
	var i int
	var needsNewline bool
	writeNewline := func() {
		if needsNewline {
			w.WriteByte('\n')
			if margin > 0 {
				w.Write(char.Repeat(' ', margin))
			}
		}
	}
	for i < len(s) {
		// Check for existing newline within termWidth
		end := min(i+width, len(s))
		if nlIdx := strings.IndexByte(s[i:end], '\n'); nlIdx >= 0 {
			// Found newline, write up to and including it
			writeNewline()
			w.WriteString(s[i : i+nlIdx+1])
			i += nlIdx + 1
			needsNewline = false
			continue
		}
		// No newline found, calculate line length
		lineLen := end - i
		// If not at end of string, try to break at last space
		if end < len(s) {
			if spaceIdx := strings.LastIndexByte(s[i:end], ' '); spaceIdx > 0 {
				lineLen = spaceIdx
			}
		}
		writeNewline()
		w.WriteString(s[i : i+lineLen])
		i += lineLen
		needsNewline = true
		// Skip spaces after break
		for i < len(s) && s[i] == ' ' {
			i++
		}
	}
}
