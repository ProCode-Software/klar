package errors

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/types"
)

// QuoteToken add quotes around source code. By default, QuoteToken uses single quotes for source code, or backticks if the source contains single quotes.
func Quote(s string) string {
	if strings.Contains(s, "'") {
		return "`" + s + "`"
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
	case lexer.EndOfStatement:
		return "newline"
	case lexer.EOF:
		return "end of file"
	}
}

func QuoteType(typ types.Type) string {
	if typ, ok := typ.(interface{ String() string }); ok {
		return Quote(typ.String())
	}
	return Quote(fmt.Sprintf("%s", typ))
}

func NameToken(tok lexer.Token) string {
	switch tok.Kind {
	default:
		return Quote(tok.Source)
	case lexer.Comma:
		return "a comma"
	case lexer.EOF:
		return "end of file"
	case lexer.EndOfStatement:
		return "a newline"
	case lexer.Colon:
		return "a colon"
	}
}

var TypeStringMap = map[lexer.TokenType]string{
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
