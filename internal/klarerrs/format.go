package klarerrs

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// QuoteToken add quotes around source code. By default, QuoteToken uses single
// quotes for source code, backticks if the source contains single quotes, or
// double quotes if it contains both or non-printable characters.
func Quote(s string) string {
	var hasSingleQuote, hasBacktick, hasCtrl bool
	for _, r := range s {
		switch {
		case r == '\'':
			hasSingleQuote = true
		case r == '`':
			hasBacktick = true
		case !unicode.IsPrint(r):
			hasCtrl = true
		}
	}
	switch {
	case hasCtrl:
		return fmt.Sprintf("%q", s)
	case !hasSingleQuote:
		return "'" + s + "'"
	case !hasBacktick:
		return "`" + s + "`"
	default:
		return `"` + s + `"`
	}
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

// FormatCount returns the given count followed by the pluralized version
// of the given word. If n == 0, "no" is used in place of the number.
func FormatCount(n int, word string) string {
	switch n {
	case 0:
		return "no " + word + "s"
	case 1:
		return "1 " + word
	default:
		return fmt.Sprintf("%d %ss", n, word)
	}
}

// FormatThis returns "this" if n is 1, or "these" otherwise.
func FormatThis(n int) string {
	if n == 1 {
		return "this"
	}
	return "these"
}

func FormatThisUpper(n int) string {
	if n == 1 {
		return "This"
	}
	return "These"
}

func FormatThisWord(n int, word string) string {
	if n == 1 {
		return "this " + word
	}
	return "these " + word + "s"
}

func Capitalize(word string) string {
	firstLetter, n := utf8.DecodeRuneInString(word)
	return string(unicode.ToUpper(firstLetter)) + word[n:]
}

func FormatCountCustom(n int, zero, one, more string) (s string) {
	// Replace %d if present. We're using this instead of fmt.Sprintf
	// because if I can recall, it will error if we pass `n` with no
	// format specifiers (extra).
	defer func() {
		if i := strings.Index(s, "%d"); i >= 0 {
			s = s[:i] + strconv.Itoa(n) + s[i+len("%d"):]
		}
	}()
	switch n {
	case 0:
		return cmp.Or(zero, more)
	case 1:
		return one
	default:
		return more
	}
}
