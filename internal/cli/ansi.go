package cli

const (
	ANSIBoldRed      = "\033[1;31m"
	ANSIBoldYellow   = "\033[1;33m"
	ANSIReset        = "\033[m"
	ANSIBoldDim      = "\033[1;2m"
	ANSIBold         = "\033[1m"
	ANSIDim          = "\033[2m"
	ANSIYellow       = "\033[33m"
	ANSIBlue         = "\033[34m"
	ANSIRed          = "\033[31m"
	ANSICyan         = "\033[36m"
	ANSIMagenta      = "\033[35m"
	ANSIGreen        = "\033[32m"
	ANSIResetBold    = "\033[0;1m"
	ANSIResetBoldDim = "\033[0;1;2m"
)

func Color(ansi string, text string) string {
	return ansi + text + ANSIReset
}
