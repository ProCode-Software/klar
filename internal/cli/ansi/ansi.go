package ansi

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

func Color(color, text string) string {
	return color + text + Reset
}
