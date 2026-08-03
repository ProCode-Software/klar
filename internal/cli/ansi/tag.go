package ansi

import (
	"fmt"
	"slices"
	"strings"
)

var ColorMap = map[string]string{
	// Foreground
	"r": "31", "g": "32", "y": "33", "b": "34",
	"m": "35", "c": "36", "w": "37",
	// Background
	"R": "41", "G": "42", "Y": "43", "B": "44",
	"M": "45", "C": "46", "W": "47",
	// Bright foreground
	"gr!": "90", "r!": "91", "g!": "92", "y!": "93",
	"b!": "94", "m!": "95", "c!": "96", "w!": "97",
	// Bright background
	"Gr!": "100", "R!": "101", "G!": "102", "Y!": "103",
	"B!": "104", "M!": "105", "C!": "106", "W!": "107",
	// Effects
	"bold": "1", "dim": "2", "ital": "3", "under": "4", "strike": "9",
	"res": "0", "reset": "0", "-": "0",
	"d": "2", "i": "3", "u": "4",
	"**": "1", "__": "1", "*": "3", "_": "3", "~~": "9", // From Markdown
}

func makeCode(colors []string) string {
	if DisableColor || len(colors) == 0 {
		return "" // So an empty slice isn't an ANSI reset
	}
	return "\x1b[" + strings.Join(colors, ";") + "m"
}

const (
	EscapeTag   = '\xff'
	UnescapeTag = '\xfe'
)

// Colorize parses the string for color tags and returns
// the string with ANSI color codes. It supports nested tags and escaping.
//
// Example: '<r **>text</>' returns text in a bold red foreground.
// See [ColorMap] for the list of available tag names/codes.
func Colorize(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	type layer struct {
		names string // The full tag content, e.g., "red bold"
		codes []string
	}
	var stack []layer
	reapply := func() {
		if DisableColor {
			return
		}
		b.WriteString("\x1b[0m")
		for _, l := range stack {
			b.WriteString(makeCode(l.codes))
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			// Escape
			if i+1 < len(s) {
				next := s[i+1]
				if next == '<' || next == '>' || next == '\\' {
					b.WriteByte(next)
					i++
					continue
				}
			}
			b.WriteByte(c)
			continue
		case EscapeTag:
			unesc := strings.IndexByte(s[i+1:], UnescapeTag)
			if unesc == -1 {
				unesc = len(s[i+1:]) // The rest of the string is escaped
			}
			b.WriteString(s[i+1 : i+1+unesc])
			unesc += i + 1 // Skip the unescape
			continue
		case UnescapeTag:
			continue // Unmatched unescape. Don't write
		case '<':
			if i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '>') {
				// '< ' and '<>', are treated as literal
				b.WriteByte(c)
				continue
			}
		default:
			b.WriteByte(c)
			continue
		}
		tagEnd := strings.IndexByte(s[i:], '>')
		if tagEnd == -1 {
			// Unterminated '<', treat everything after as literal
			b.WriteString(s[i:])
			break
		}
		tagEnd += i
		tagContent := s[i+1 : tagEnd]
		// Closing tag
		if colorsToClose, ok := strings.CutPrefix(tagContent, "/"); ok {
			i = tagEnd
			colorsToClose = strings.TrimSpace(colorsToClose)
			if colorsToClose == "" {
				// </> closes everything
				stack = stack[:0]
				reapply()
				continue
			}
			// </...> may partially close the top of the stack (but not anything
			// opened before). `<** r>...</r>` is allowed.
			top := stack[len(stack)-1]
			for colorName := range strings.FieldsSeq(colorsToClose) {
				code, ok := ColorMap[colorName]
				if !ok {
					panic(fmt.Sprintf("Colorize: unknown color %q", colorName))
				}
				inStackI := slices.Index(top.codes, code)
				if inStackI < 0 {
					panic(fmt.Sprintf(
						"Colorize: at position %d, %q has not been opened most recently",
						i, colorName,
					))
				}
				top.codes = fastDelete(top.codes, inStackI)
			}
			if len(top.codes) == 0 {
				stack = stack[:len(stack)-1]
			}
			reapply()
			continue
		}
		// Opening tag
		parts := strings.Fields(tagContent) // Split by whitespace
		if len(parts) == 0 {
			// Empty tag <>, treat as literal
			b.WriteByte(c)
			continue
		}
		codes := make([]string, len(parts))
		for i, part := range parts {
			code, ok := ColorMap[part]
			if !ok {
				panic(fmt.Sprintf("Colorize: unknown color %q", tagContent))
			}
			codes[i] = code
		}
		stack = append(stack, layer{names: tagContent, codes: codes})
		b.WriteString(makeCode(codes))
		i = tagEnd
	}
	return b.String()
}

// Here because importing internal/util creates a cycle
func fastDelete[T any](s []T, i int) []T {
	_ = s[i] // Bounds check
	if len(s) == 1 {
		s = s[:0]
		return s
	}
	last := len(s) - 1
	s[i], s[last] = s[last], s[i]
	s = s[:last]
	return s
}
