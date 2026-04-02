package cli

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/char"
)

type Wrapper struct {
	Width, FirstRowWidth int
	AllWriter
	Margin       int
	WriteNewline func()
}

func Wrap(s string, w AllWriter, width, firstRowWith, margin int) {
	wr := &Wrapper{
		Width:         width,
		FirstRowWidth: firstRowWith,
		AllWriter:     w,
		Margin:        margin,
	}
	wr.Wrap(s)
}

// Wrap wraps a string to a given width, with an optional left margin
// and first row width. The first row is unindented.
func (wr *Wrapper) Wrap(s string) {
	if wr.Width <= 0 {
		wr.WriteString(s)
		return
	}
	var (
		textWidth    = max(wr.Width-wr.Margin, 1)
		i            int
		needsNewline bool
		currWidth    = wr.Width
	)
	if wr.FirstRowWidth > 0 {
		currWidth = wr.FirstRowWidth
	}
	writeNewline := func() {
		if needsNewline {
			wr.WriteByte('\n')
			if wr.Margin > 0 {
				wr.Write(char.Repeat(' ', wr.Margin))
			}
			if wr.WriteNewline != nil {
				wr.WriteNewline()
			}
		}
	}
	for i < len(s) {
		// Calculate the end of the line based on visual width, skipping ANSI escapes
		byteEnd, hasNewline := findVisibleEnd(s[i:], currWidth)
		if hasNewline {
			// Handle explicit newlines in the input string
			newlineI := strings.IndexByte(s[i:i+byteEnd+1], '\n')
			writeNewline()
			wr.WriteString(s[i : i+newlineI+1])
			i += newlineI + 1
			if wr.Margin > 0 && i < len(s) {
				wr.Write(char.Repeat(' ', wr.Margin))
			}
			needsNewline = false // Newline already written
			currWidth = textWidth
			continue
		}
		// Wrap at the current visual width limit
		actualEnd := i + byteEnd
		if i+byteEnd < len(s) {
			// Try to break at the last space within the width to avoid splitting words
			if spaceI := strings.LastIndexByte(s[i:i+byteEnd], ' '); spaceI > 0 {
				actualEnd = i + spaceI
			} else {
				// No space found within the width; find the next space/newline to avoid cutting the word
				nextBreak := strings.IndexAny(s[i+byteEnd:], " \n")
				if nextBreak >= 0 {
					actualEnd = i + byteEnd + nextBreak
				} else {
					actualEnd = len(s) // End of string
				}
			}
		}
		writeNewline()
		wr.WriteString(s[i:actualEnd])
		i = actualEnd
		needsNewline = true
		currWidth = textWidth
		// Skip spaces trailing the wrap point
		for i < len(s) && s[i] == ' ' {
			i++
		}
	}
}

// findVisibleEnd finds the byte offset of the end of the line within the given width,
// treating each non-ANSI byte as a single visual character and stopping at newlines.
func findVisibleEnd(s string, width int) (byteEnd int, hasNewline bool) {
	visible := 0
	for byteEnd < len(s) && visible < width {
		if s[byteEnd] == '\n' {
			return byteEnd, true
		}
		// Skip ANSI escape sequences without counting them as visible
		if s[byteEnd] == '\x1b' {
			byteEnd += skipANSI(s[byteEnd:])
			continue
		}
		byteEnd++
		visible++
	}
	return byteEnd, false
}

// skipANSI includes the '\x1b' byte in the returned length.
func skipANSI(s string) (n int) {
	if len(s) < 2 || s[0] != '\x1b' {
		return 0
	}
	n++
	var ansiEndMin byte = 0x30
	start := 1
	if s[1] == '[' {
		ansiEndMin, start = 0x40, 2
		n++
	}
	for j := start; j < len(s); j++ {
		c := s[j]
		n++
		if c >= ansiEndMin && c <= ansiEndMax {
			return n
		}
	}
	return len(s)
}
