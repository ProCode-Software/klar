package errors

import "strings"

type KlarError interface {
	error
	KlarError()
}

type ErrorCode int

func (ParseError) KlarError() {}

// Quotes add quotes around source code. By default, Quote uses single quotes for source code, or backticks if the source contains single quotes.
func Quote(src string) string {
	switch {
	default:
		return "'" + src + "'"
	case strings.Contains(src, "'"):
		return "`" + src + "`"
	case src == "\n":
		return "end of statement"
	}
}

// QuoteToken returns 'token X', where X is the quoted source code, or 'end of statement' if EOS.
func QuoteToken(src string) string {
	if src == "\n" {
		return "end of statement"
	}
	return "token" + Quote(src)
}
