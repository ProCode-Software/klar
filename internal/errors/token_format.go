package errors

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
)

func QuoteString(s string) string {
	if strings.Contains(s, "'") {
		return "`" + s + "`"
	}
	return "'" + s + "'"
}

// Quotes add quotes around source code. By default, Quote uses single quotes for source code, or backticks if the source contains single quotes.
func Quote(tok lexer.Token) string {
	switch {
	default:
		return QuoteString(tok.Source)
	case tok.Kind == lexer.EndOfStatement:
		return "a newline"
	case tok.Kind == lexer.EOF:
		return "end of file"
	}
}

func QuoteWithoutA(tok lexer.Token) string {
	return strings.TrimPrefix(Quote(tok), "a ")
}

// QuoteToken returns 'token X', where X is the quoted source code, or 'end of statement' if EOS.
func QuoteToken(tok lexer.Token) string {
	switch tok.Kind {
	default:
		return "token " + Quote(tok)
	case lexer.EndOfStatement:
		return "a newline"
	case lexer.EOF:
		return "end of file"
	case lexer.Comma:
		return "a comma"
	case lexer.Colon:
		return "a colon"
	case lexer.Dot:
		return "a period"
	}
}

func QuoteTokenThis(tok lexer.Token) string {
	return strings.Replace(QuoteToken(tok), "a ", "this ", 1)
}

var vowels = map[byte]bool{
	'A': true, 'E': true, 'I': true, 'O': true, 'U': true,
	'a': true, 'e': true, 'i': true, 'o': true, 'u': true,
}

func FormatTokenType(tok lexer.TokenType) string {
	switch tok {
	default:
		for src, kind := range lexer.OperatorMap {
			if kind == tok {
				return Quote(lexer.Token{Source: src, Kind: kind})
			}
		}
		if vowels[tok.String()[0]] {
			return "an " + tok.String()
		}
		return "a " + tok.String()
	case lexer.EndOfStatement:
		return "a newline"
	case lexer.EOF:
		return "end of file"
	case lexer.Comma:
		return "a comma"
	case lexer.Colon:
		return "a colon"
	case lexer.Dot:
		return "a period"
	case 0:
		return "<unknown>"
	}
}
