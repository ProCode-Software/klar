package ansi

import "os"

const (
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Red     = "\033[31m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
	Green   = "\033[32m"

	Reset     = "\033[m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	BoldRed      = "\033[1;31m"
	BoldYellow   = "\033[1;33m"
	BoldDim      = "\033[1;2m"
	ResetBold    = "\033[0;1m"
	ResetBoldDim = "\033[0;1;2m"
)

var isNoColor bool

func init() {
	isNoColor = os.Getenv("NO_COLOR") != ""
}

func Color(color, text string) string {
	return color + text + Reset
}

type Colors struct {
	NoColor bool
}

func (c Colors) Red(s string) string          { return c.color(Red, s) }
func (c Colors) Yellow(s string) string       { return c.color(Yellow, s) }
func (c Colors) Blue(s string) string         { return c.color(Blue, s) }
func (c Colors) Green(s string) string        { return c.color(Green, s) }
func (c Colors) Cyan(s string) string         { return c.color(Cyan, s) }
func (c Colors) Magenta(s string) string      { return c.color(Magenta, s) }
func (c Colors) Bold(s string) string         { return c.color(Bold, s) }
func (c Colors) Dim(s string) string          { return c.color(Dim, s) }
func (c Colors) Italic(s string) string       { return c.color(Italic, s) }
func (c Colors) Underline(s string) string    { return c.color(Underline, s) }
func (c Colors) BoldRed(s string) string      { return c.color(BoldRed, s) }
func (c Colors) BoldYellow(s string) string   { return c.color(BoldYellow, s) }
func (c Colors) BoldDim(s string) string      { return c.color(BoldDim, s) }
func (c Colors) ResetBold(s string) string    { return c.color(ResetBold, s) }
func (c Colors) ResetBoldDim(s string) string { return c.color(ResetBoldDim, s) }
func (c Colors) Reset() string                { return c.color(Reset, "") }

func (c Colors) color(color, s string) string {
	if c.NoColor || isNoColor {
		return s
	}
	return color + s + Reset
}
