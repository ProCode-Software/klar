package klarerrs

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// QuoteToken add quotes around source code. By default, QuoteToken uses single
// quotes for source code, or backticks if the source contains single quotes.
func Quote(s string) string {
	if strings.Contains(s, "'") {
		return "`" + s + "`"
	}
	if len(s) > 0 && !unicode.IsPrint(rune(s[0])) {
		return fmt.Sprintf("%q", s)
	}
	return "'" + s + "'"
}

func QuoteToken(tok lexer.Token) string {
	switch tok.Kind {
	default:
		return Quote(tok.Source)
	case lexer.Comma:
		return "comma"
	case lexer.Colon:
		return "colon"
	case lexer.Newline:
		return "newline"
	case lexer.EOF:
		return "end of file"
	}
}

func NameToken(tok lexer.Token) string {
	if str, ok := TypeStringMap[tok.Kind]; ok {
		return str
	}
	return Quote(tok.Source)
}

var TypeStringMap = map[lexer.TokenType]string{
	lexer.Identifier: "an identifier",
	lexer.Numeric:    "a number",
	lexer.Boolean:    "a boolean",
	lexer.String:     "a string",
	lexer.Regex:      "a regular expression",
	lexer.Nil:        "'nil'",
	lexer.And:        "'and'",
	lexer.Or:         "'or'",
	lexer.Newline:    "a newline",
	lexer.EOF:        "end of file",
	lexer.Comma:      "a comma",
	lexer.Colon:      "a colon",
	lexer.Dot:        "a period",
	0:                "<unknown>",
}

func init() {
	for str, kw := range lexer.KeywordMap {
		TypeStringMap[kw] = "'" + str + "'"
	}
	for str, op := range lexer.OperatorMap {
		TypeStringMap[op] = "'" + str + "'"
	}
}

func FormatTokenType(tok lexer.TokenType) string {
	if s, ok := TypeStringMap[tok]; ok {
		return s
	}
	panic(fmt.Sprintf("cannot represent token type %s as string", tok))
}

func WithA(str string) string {
	switch str[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an " + str
	}
	return "a " + str
}

// Format returns code as a camelCase string.
func (c Code) Format() string {
	str := c.String()
	str = strings.TrimPrefix(str, "Err")
	first := unicode.ToLower(rune(str[0]))
	return string(first) + str[1:]
}
