package ansi

import (
	"fmt"
	"io"
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
	"bold": "1", "dim": "2", "ital": "3", "under": "4", "res": "0", "**": "1", "-": "0",
	"i": "3", "u": "4", "d": "2",
}

func makeCode(colors []string) string {
	if DisableColor || len(colors) == 0 {
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
		tagName string // The full tag content, e.g., "red bold"
		codes   []string
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
	var disable bool
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
			disable = true
			continue // Don't write
		case UnescapeTag:
			disable = false
			continue // Don't write
		case '<':
			if i+1 < len(s) && s[i+1] == ' ' {
				// If '<' is followed by a space, treat as literal
				b.WriteByte(c)
				continue
			}
		default:
			b.WriteByte(c)
			continue
		}
		if disable {
			b.WriteByte(c)
			continue
		}
		end := strings.IndexByte(s[i:], '>')
		if end == -1 {
			// Unterminated '<', treat as literal
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
			for _, part := range parts {
				code, ok := ColorMap[part]
				if !ok {
					panic(fmt.Sprintf("ansi: unknown color %q", tagName))
				}
				closingCodes = append(closingCodes, code)
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
		for _, part := range parts {
			code, ok := ColorMap[part]
			if !ok {
				panic(fmt.Sprintf("ansi: unknown color %q", tagContent))
			}
			codes = append(codes, code)
		}
		l := layer{tagName: tagContent, codes: codes}
		stack = append(stack, l)
		b.WriteString(makeCode(codes))
		i = end
	}
	return b.String()
}

func EscapeFromColorize(s string) string {
	return string(EscapeTag) + s + string(UnescapeTag)
}

// Decolorize strips tags from v, returning an uncolorized string without tags.
func Decolorize(v string) string {
	currDisableColor := DisableColor
	DisableColor = true
	s := Colorize(v)
	DisableColor = currDisableColor
	return s
}

func TagFprintf(w io.Writer, format string, a ...any) (n int, err error) {
	return fmt.Fprintf(w, Colorize(format), a...)
}

func TagFprintfln(w io.Writer, format string, a ...any) (n int, err error) {
	return fmt.Fprintf(w, Colorize(format)+"\n", a...)
}

func Colorizef(format string, a ...any) string {
	return Colorize(fmt.Sprintf(format, a...))
}

func TagPrintln(v string) (n int, err error) {
	return fmt.Println(Colorize(v))
}

func TagPrintfln(format string, a ...any) (n int, err error) {
	return fmt.Println(Colorizef(format, a...))
}

// color is an color code. v is the raw ANSI string. every time v has an ansi reset,
// code is reapplied.
func Wrap(code string, v string) string {
	// TODO
	return code + v + CodeReset
}
