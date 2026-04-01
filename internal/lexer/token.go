package lexer

import (
	"fmt"
	"io"
)

type Token struct {
	Position
	Kind       TokenType
	Source     string
	Attributes map[string]any
}

type attrs = map[string]any

func NewToken(pos Position, kind TokenType, src string) *Token {
	return &Token{pos, kind, src, nil}
}

func (t *Token) Len() uint32 {
	if t.Attributes != nil {
		if v, ok := t.Attributes["length"].(uint32); ok {
			return v
		}
	}
	return uint32(len(t.Source))
}

func (t *Token) End() Position {
	if t.Attributes != nil {
		if end, ok := t.Attributes["end"].(Position); ok && end.Line > 0 {
			return end
		}
	}
	return Position{Line: t.Position.Line, Col: t.Position.Col + t.Len()}
}

func (t Token) String() string {
	s := fmt.Sprintf("Token{%s %s: %#q", t.Position, t.Kind, t.Source)
	if t.Attributes != nil {
		s += fmt.Sprintf(" %+v", t.Attributes)
	}
	return s + "}"
}

func (t *Token) setAttr(key string, value any) *Token {
	if t.Attributes == nil {
		t.Attributes = make(attrs)
	}
	t.Attributes[key] = value
	return t
}

func (t *Token) withAttrs(attrs attrs) *Token {
	t.Attributes = attrs
	return t
}

func (t TokenType) LitterDump(w io.Writer) {
	w.Write([]byte("{" + t.String() + "}"))
}

var TokenTypeString = map[TokenType]string{
	0:          "<unknown>",
	String:     "string",
	Numeric:    "number",
	Newline:    "newline",
	Illegal:    "illegal",
	Identifier: "identifier",
	Regex:      "regular expression",
	EOF:        "EOF",
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

func (k TokenType) String() string { return TokenTypeString[k] }

// A map of the begin character in an operator and how many bytes
// to read after to parse an operator
var opPrefixes = make(map[rune]int, len(OperatorMap)/2)

func init() {
	for op := range OperatorMap {
		first := rune(op[0])
		opPrefixes[first] = max(opPrefixes[first], len(op)-1)
	}
}
