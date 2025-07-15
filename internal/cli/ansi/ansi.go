package ansi

import "os"

const (
	CodeYellow  = "\033[33m"
	CodeBlue    = "\033[34m"
	CodeRed     = "\033[31m"
	CodeCyan    = "\033[36m"
	CodeMagenta = "\033[35m"
	CodeGreen   = "\033[32m"

	CodeReset     = "\033[m"
	CodeBold      = "\033[1m"
	CodeDim       = "\033[2m"
	CodeItalic    = "\033[3m"
	CodeUnderline = "\033[4m"

	CodeBoldRed      = "\033[1;31m"
	CodeBoldYellow   = "\033[1;33m"
	CodeBoldGreen    = "\033[1;32m"
	CodeBoldDim      = "\033[1;2m"
	CodeResetBold    = "\033[0;1m"
	CodeResetBoldDim = "\033[0;1;2m"
)

var NoColor = os.Getenv("NO_COLOR") != ""

func Color(color, text string) string {
	if NoColor {
		return text
	}
	return color + text + CodeReset
}

type Colors struct {
	NoColor bool
}

func Red(s string) string          { return Color(CodeRed, s) }
func Yellow(s string) string       { return Color(CodeYellow, s) }
func Blue(s string) string         { return Color(CodeBlue, s) }
func Green(s string) string        { return Color(CodeGreen, s) }
func Cyan(s string) string         { return Color(CodeCyan, s) }
func Magenta(s string) string      { return Color(CodeMagenta, s) }
func Bold(s string) string         { return Color(CodeBold, s) }
func Dim(s string) string          { return Color(CodeDim, s) }
func Italic(s string) string       { return Color(CodeItalic, s) }
func Underline(s string) string    { return Color(CodeUnderline, s) }
func BoldRed(s string) string      { return Color(CodeBoldRed, s) }
func BoldYellow(s string) string   { return Color(CodeBoldYellow, s) }
func BoldGreen(s string) string    { return Color(CodeBoldGreen, s) }
func BoldDim(s string) string      { return Color(CodeBoldDim, s) }
func ResetBold(s string) string    { return Color(CodeResetBold, s) }
func ResetBoldDim(s string) string { return Color(CodeResetBoldDim, s) }
func Reset() string                { return Color(CodeReset, "") }
