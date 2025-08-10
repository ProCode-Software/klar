package lexer

import (
	"fmt"
	"io"
)

const (
	NumberFormatDecimal = iota
	NumberFormatHex
	NumberFormatOctal
	NumberFormatBinary
)

func NewToken(pos Position, kind TokenType, src string) *Token {
	return &Token{pos, kind, src, nil}
}

type Token struct {
	Position
	Kind       TokenType
	Source     string
	Attributes map[string]any
}

func (t *Token) SetAttribute(key string, value any) *Token {
	if t.Attributes == nil {
		t.Attributes = make(map[string]any)
	}
	t.Attributes[key] = value
	return t
}

func (t TokenType) LitterDump(w io.Writer) {
	w.Write([]byte("{" + t.String() + "}"))
}

func (t Token) String() string {
	s := fmt.Sprintf("%s %s: %#q", t.Position, t.Kind, t.Source)
	if t.Attributes != nil {
		s += fmt.Sprintf(" %+v", t.Attributes)
	}
	return s
}

func (p Position) LitterDump(w io.Writer) {
	w.Write([]byte("{" + p.String() + "}"))
}

var TokenTypeString = map[TokenType]string{
	0:              "<unknown>",
	String:         "string",
	Numeric:        "number",
	Newline:        `\n`,
	EndOfStatement: "newline",
	Illegal:        "illegal",
	Identifier:     "identifier",
	EOF:            "EOF",
}

func init() {
	for str, t := range OperatorMap {
		TokenTypeString[t] = str
	}
	for str, t := range KeywordMap {
		TokenTypeString[t] = str
	}
	TokenTypeString[Boolean] = "boolean"
}

func (k TokenType) String() string {
	return TokenTypeString[k]
}

// A map of the begin character in an operator and how many bytes
// to read after to parse an operator
var opPrefixes = make(map[rune]int, len(OperatorMap)/2)

func init() {
	for op := range OperatorMap {
		first := rune(op[0])
		opPrefixes[first] = max(opPrefixes[first], len(op)-1)
	}
}
