package ansi

import (
	"fmt"
	"io"
	"strings"
)

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

// Wrap is equivalent to [Color](color, s), but color is reapplied
// whenever s has an ANSI reset.
func Wrap(color string, s string) string {
	if DisableColor {
		return s
	}
	return color + strings.NewReplacer(
		"\x1b[m", "\x1b[;"+color+"m",
		"\x1b[0m", "\x1b[0;"+color+"m",
		"\x1b[0;", "\x1b[0;"+color+";",
		"\x1b[;", "\x1b["+color+";",
	).Replace(s)
}
