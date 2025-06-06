package errors

import (
	"fmt"
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
	switch tok.Kind {
	default:
		return QuoteString(tok.Source)
	case lexer.Comma:
		return "comma"
	case lexer.Colon:
		return "colon"
	case lexer.EndOfStatement:
		return "newline"
	case lexer.EOF:
		return "end of file"
	}
}

func QuoteA(tok lexer.Token) string {
	switch tok.Kind {
	default:
		return QuoteString(tok.Source)
	case lexer.Comma:
		return "a comma"
	case lexer.EndOfStatement:
		return "a newline"
	case lexer.Colon:
		return "a colon"
	}
}

var vowels = map[byte]bool{
	'A': true, 'E': true, 'I': true, 'O': true, 'U': true,
	'a': true, 'e': true, 'i': true, 'o': true, 'u': true,
}

var stringMap = map[lexer.TokenType]string{
	lexer.Identifier:     "an identifier",
	lexer.Numeric:        "a number",
	lexer.Boolean:        "a boolean",
	lexer.String:         "a string",
	lexer.Nil:            "'nil'",
	lexer.And:            "'and'",
	lexer.Or:             "'or'",
	lexer.EndOfStatement: "a newline",
	lexer.EOF:            "end of file",
	lexer.Comma:          "a comma",
	lexer.Colon:          "a colon",
	lexer.Dot:            "a period",
	0:                    "<unknown>",
}

func init() {
	for str, kw := range lexer.KeywordMap {
		stringMap[kw] = "'" + str + "'"
	}
	for str, op := range lexer.OperatorMap {
		stringMap[op] = "'" + str + "'"
	}
}

func FormatTokenType(tok lexer.TokenType) string {
	if s, ok := stringMap[tok]; ok {
		return s
	}
	panic(fmt.Sprintf("cannot represent token type %s as string", tok))
}
