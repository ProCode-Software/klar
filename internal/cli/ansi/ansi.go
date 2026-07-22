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
	CodeBoldCyan     = "\033[1;36m"
	CodeBoldDim      = "\033[1;2m"
	CodeBoldBlue     = "\033[1;34m"
	CodeResetBold    = "\033[0;1m"
	CodeResetBoldDim = "\033[0;1;2m"

	CodeBrightWhite   = "\033[97m"
	CodeBrightRed     = "\033[91m"
	CodeBrightGreen   = "\033[92m"
	CodeBrightYellow  = "\033[93m"
	CodeBrightBlue    = "\033[94m"
	CodeBrightMagenta = "\033[95m"
	CodeBrightCyan    = "\033[96m"

	CodeBoldBrightWhite   = "\033[1;97m"
	CodeBoldBrightRed     = "\033[1;91m"
	CodeBoldBrightYellow  = "\033[1;93m"
	CodeBoldBrightGreen   = "\033[1;92m"
	CodeBoldBrightBlue    = "\033[1;94m"
	CodeBoldBrightMagenta = "\033[1;95m"
	CodeBoldBrightCyan    = "\033[1;96m"

	CodeDimGreen   = "\033[2;32m"
	CodeDimCyan    = "\033[2;36m"
	CodeDimBlue    = "\033[2;34m"
	CodeDimMagenta = "\033[2;35m"

	ClearLine = "\033[2K"
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
func BoldCyan(s string) string    { return Color(CodeBoldCyan, s) }
func BoldBlue(s string) string    { return Color(CodeBoldBlue, s) }
func BoldDim(s string) string     { return Color(CodeBoldDim, s) }
func DimCyan(s string) string     { return Color(CodeDimCyan, s) }
func DimBlue(s string) string     { return Color(CodeDimBlue, s) }
func DimMagenta(s string) string  { return Color(CodeDimMagenta, s) }
func DimGreen(s string) string    { return Color(CodeDimGreen, s) }

func BrightRed(s string) string     { return Color(CodeBrightRed, s) }
func BrightGreen(s string) string   { return Color(CodeBrightGreen, s) }
func BrightYellow(s string) string  { return Color(CodeBrightYellow, s) }
func BrightBlue(s string) string    { return Color(CodeBrightBlue, s) }
func BrightMagenta(s string) string { return Color(CodeBrightMagenta, s) }
func BrightCyan(s string) string    { return Color(CodeBrightCyan, s) }

func BoldBrightWhite(s string) string   { return Color(CodeBoldBrightWhite, s) }
func BoldBrightRed(s string) string     { return Color(CodeBoldBrightRed, s) }
func BoldBrightGreen(s string) string   { return Color(CodeBoldBrightGreen, s) }
func BoldBrightYellow(s string) string  { return Color(CodeBoldBrightYellow, s) }
func BoldBrightBlue(s string) string    { return Color(CodeBoldBrightBlue, s) }
func BoldBrightMagenta(s string) string { return Color(CodeBoldBrightMagenta, s) }
func BoldBrightCyan(s string) string    { return Color(CodeBoldBrightCyan, s) }

func Gradient(text string, colors ...[3]int) string {
	return gradient("38", text, colors...)
}

// space is either "38" (foreground) or "48" (background).
func RGBSpace(space string, r, g, b int) string {
	return fmt.Sprintf("\033[%s;2;%d;%d;%dm", space, r, g, b)
}

var formatRegex = regexp.MustCompile(`(%[]\[#+.0-9]*[A-Za-z])`)

func ColorSprintf(color, format string, a ...any) string {
	if DisableColor {
		return fmt.Sprintf(format, a...)
	}
	new := formatRegex.ReplaceAllString(format, "$1"+CodeReset+color)
	return fmt.Sprintf(color+new, a...) + CodeReset
}

func ColorPrintfln(color, format string, a ...any) {
	fmt.Println(ColorSprintf(color, format, a...))
}

func ColorPrintln(color string, v ...any) {
	if DisableColor {
		fmt.Println(v...)
		return
	}
	fmt.Print(color)
	fmt.Print(v...)
	fmt.Println(CodeReset)
}

func ColorFprintln(file io.Writer, color, format string, a ...any) {
	fmt.Fprintln(file, ColorSprintf(color, format, a...))
}

/*
isattyIn := term.IsTerminal(int(os.Stdin.Fd()))
isattyOut := term.IsTerminal(int(os.Stdout.Fd()))
term := os.Getenv("TERM")
noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("NO_COLOR") == "1"
useColor := isattyOut && term != "" && term != "dumb" && !noColor
*/

func Hyperlink(label, url string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, label)
}
