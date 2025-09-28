package ansi

import (
	"fmt"
	"io"
	"os"
	"regexp"
)

const (
	CodeYellow  = "\033[33m"
	CodeBlue    = "\033[34m"
	CodeRed     = "\033[31m"
	CodeCyan    = "\033[36m"
	CodeMagenta = "\033[35m"
	CodeGreen   = "\033[32m"
	CodeGray    = "\033[90m"

	CodeReset     = "\033[m"
	CodeBold      = "\033[1m"
	CodeDim       = "\033[2m"
	CodeItalic    = "\033[3m"
	CodeUnderline = "\033[4m"

	CodeBoldRed      = "\033[1;31m"
	CodeBoldYellow   = "\033[1;33m"
	CodeBoldGreen    = "\033[1;32m"
	CodeBoldMagenta  = "\033[1;35m"
	CodeBoldDim      = "\033[1;2m"
	CodeBoldBlue     = "\033[1;34m"
	CodeResetBold    = "\033[0;1m"
	CodeResetBoldDim = "\033[0;1;2m"

	CodeBrightWhite     = "\033[97m"
	CodeBrightRed       = "\033[91m"
	CodeBoldBrightWhite = "\033[1;97m"

	CodeDimCyan = "\033[2;36m"
)

var DisableColor = os.Getenv("NO_COLOR") != ""

func Color(color, text string) string {
	if DisableColor || color == "" {
		return text
	}
	return color + text + CodeReset
}

func Partial(color string) string {
	if DisableColor {
		return ""
	}
	return color
}

type Colors struct {
	NoColor bool
}

func Reset() string               { return Color(CodeReset, "") }
func Red(s string) string         { return Color(CodeRed, s) }
func Yellow(s string) string      { return Color(CodeYellow, s) }
func Blue(s string) string        { return Color(CodeBlue, s) }
func Green(s string) string       { return Color(CodeGreen, s) }
func Cyan(s string) string        { return Color(CodeCyan, s) }
func Magenta(s string) string     { return Color(CodeMagenta, s) }
func Bold(s string) string        { return Color(CodeBold, s) }
func Dim(s string) string         { return Color(CodeDim, s) }
func Gray(s string) string        { return Color(CodeGray, s) }
func Italic(s string) string      { return Color(CodeItalic, s) }
func Underline(s string) string   { return Color(CodeUnderline, s) }
func BoldRed(s string) string     { return Color(CodeBoldRed, s) }
func BoldYellow(s string) string  { return Color(CodeBoldYellow, s) }
func BoldGreen(s string) string   { return Color(CodeBoldGreen, s) }
func BoldMagenta(s string) string { return Color(CodeBoldMagenta, s) }
func BoldBlue(s string) string    { return Color(CodeBoldBlue, s) }
func BoldDim(s string) string     { return Color(CodeBoldDim, s) }
func DimCyan(s string) string     { return Color(CodeDimCyan, s) }
func BrightRed(s string) string   { return Color(CodeBrightRed, s) }

func BoldBrightWhite(s string) string { return Color(CodeBoldBrightWhite, s) }

var formatRegex = regexp.MustCompile(`(%[]\[#+.0-9]*[A-Za-z])`)

func Sprintf(color, format string, a ...any) string {
	if DisableColor {
		return fmt.Sprintf(format, a...)
	}
	new := formatRegex.ReplaceAllString(format, "$1"+CodeReset+color)
	return fmt.Sprintf(color+new, a...) + CodeReset
}

func Println(color, format string, a ...any) {
	fmt.Println(Sprintf(color, format, a...))
}

func Fprintln(file io.Writer, color, format string, a ...any) {
	fmt.Fprintln(file, Sprintf(color, format, a...))
}
