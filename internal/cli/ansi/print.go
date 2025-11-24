package ansi

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

var colorMap = map[string]string{
	// Foreground
	"red": "31", "green": "32", "yellow": "33", "blue": "34",
	"magenta": "35", "cyan": "36", "white": "37",
	// Background
	"Red": "41", "Green": "42", "Yellow": "43", "Blue": "44",
	"Magenta": "45", "Cyan": "46", "White": "47",
	// Bright foreground
	"gray": "90", "brightRed": "91", "brightGreen": "92", "brightYellow": "93",
	"brightBlue": "94", "brightMagenta": "95", "brightCyan": "96", "brightWhite": "97",
	// Bright background
	"Gray": "100", "BrightRed": "101", "BrightGreen": "102", "BrightYellow": "103",
	"BrightBlue": "104", "BrightMagenta": "105", "BrightCyan": "106", "BrightWhite": "107",
	// Effects
	"bold": "1", "dim": "2", "italic": "3", "underline": "4", "reset": "0",
}

func makeCode(colors []string) string {
	if len(colors) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\x1b[")
	for i, code := range colors {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(code)
	}
	b.WriteByte('m')
	return b.String()
}

// Colorize parses the string for color tags (e.g. <red bold>text</>) and returns
// the string with ANSI color codes. It supports nested tags and escaping.
func Colorize(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	type layer struct {
		tagName string // The full tag content, e.g., "red bold"
		codes   []string
	}
	var stack []layer

	reapply := func() {
		b.WriteString("\x1b[0m")
		for _, l := range stack {
			b.WriteString(makeCode(l.codes))
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		// Handle escaping
		if c == '\\' {
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
		}

		if c != '<' {
			b.WriteByte(c)
			continue
		}

		// Potential tag start
		end := strings.IndexByte(s[i:], '>')
		if end == -1 {
			b.WriteByte(c)
			continue
		}
		end += i

		tagContent := s[i+1 : end]

		// Closing tag
		if after, ok := strings.CutPrefix(tagContent, "/"); ok {
			i = end
			if len(stack) == 0 {
				continue
			}

			tagName := strings.TrimSpace(after)
			if tagName == "" {
				// Generic close </>, pop top
				stack = stack[:len(stack)-1]
				reapply()
				continue
			}

			// Named close </name>
			parts := strings.Fields(tagName)
			if len(parts) == 0 {
				// Treated as generic close
				stack = stack[:len(stack)-1]
				reapply()
				continue
			}

			// Parse colors
			var closingCodes []string
			validClose := true
			for _, part := range parts {
				code, ok := colorMap[part]
				if !ok {
					validClose = false
					break
				}
				closingCodes = append(closingCodes, code)
			}

			if !validClose {
				continue
			}

			top := stack[len(stack)-1]
			allFound := true
			for _, closeCode := range closingCodes {
				if !slices.Contains(top.codes, closeCode) {
					allFound = false
					break
				}
			}

			if allFound {
				stack = stack[:len(stack)-1]
				reapply()
			}
			continue
		}

		// Opening tag
		parts := strings.Fields(tagContent) // Split by whitespace
		if len(parts) == 0 {
			// Empty tag <>, treat as literal
			b.WriteByte(c)
			continue
		}

		var codes []string
		valid := true
		for _, part := range parts {
			code, ok := colorMap[part]
			if !ok {
				valid = false
				break
			}
			codes = append(codes, code)
		}

		if valid {
			l := layer{tagName: tagContent, codes: codes}
			stack = append(stack, l)
			b.WriteString(makeCode(codes))
			i = end
		} else {
			// Not a valid color tag, treat as literal
			b.WriteByte(c)
		}
	}

	// If stack is not empty at the end, reset
	if len(stack) > 0 {
		b.WriteString("\x1b[0m")
	}

	return b.String()
}

func Fprintf(w io.Writer, format string, a ...any) (n int, err error) {
	return fmt.Fprintf(w, Colorize(format), a...)
}

func Sprintf(format string, a ...any) string {
	return fmt.Sprintf(Colorize(format), a...)
}

func Println(v string) (n int, err error) {
	return fmt.Println(Colorize(v))
}
