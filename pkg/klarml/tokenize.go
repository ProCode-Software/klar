package klarml

import (
	"strings"
	"unicode"
)

type lexer struct {
	Bytes []byte
	Index int
	Position
}

func (l *lexer) Next() byte {
	current := l.Current()
	l.Index++
	if l.Bytes[l.Index] == '\n' {
		l.Line++
		l.Col = 0
	}
	l.Col++
	return current
}

func (l *lexer) Backup() {
	l.Col--
	l.Index--
}

func newToken(pos Position, kind TokenType, src string) Token {
	return Token{Position: pos, Kind: kind, Source: src}
}

func (l *lexer) Peek() byte     { return l.Bytes[l.Index+1] }
func (l *lexer) Current() byte  { return l.Bytes[l.Index] }
func (l *lexer) HasBytes() bool { return l.Index < len(l.Bytes) }

var wasNewline bool

func (l *lexer) Tokenize() Token {
	pos := l.Position
	b := l.Next()
	defer func() { wasNewline = false }()
	switch b {
	case ':':
		return newToken(pos, Colon, ":")
	case '-':
		return newToken(pos, Hyphen, "-")
	case '{':
		return newToken(pos, LeftBrace, "{")
	case '}':
		return newToken(pos, RightBrace, "}")
	case '.':
		return newToken(pos, Period, ".")
	case '$':
		return newToken(pos, Dollar, "$")
	case '\n':
		wasNewline = true
		return newToken(pos, Newline, "\n")
	case '=':
		return newToken(pos, Equal, "=")
	case '@':
		return l.ParseNamespace(pos)
	case '/':
		next := l.Peek()
		if next == '/' || next == '*' {
			l.Next()
			return l.ParseComment(pos, next == '*')
		}
	case '"', '\'':
		return l.ParseString(pos, b)
	}
	r := rune(b)
	switch {
	case unicode.IsLetter(r):
		return l.ParseIdent(pos, b)
	case unicode.IsDigit(r), r == '_', r == '+', r == '.':
		return l.ParseIdent(pos, b)
	case unicode.IsSpace(r):
		if wasNewline {
			return l.Tokenize()
		}
		fallthrough
	default:
		return l.ParseUnquoted(pos, b)
	}
}

var allowedInVar = map[rune]bool{
	'.': true, '+': true, '-': true, '_': true,
}

func (l *lexer) ParseNamespace(start Position) Token {
	ident := l.ParseIdent(l.Position, '@').Source
	return newToken(start, Namespace, ident)
}

// var: /(\$)(\.?[-\p{L}\w_]+)/u,
// ns: /@[\p{L}\w\d_.+-]+/u
// key: /[-\p{L}\w._/+$@]+/
func (l *lexer) ParseString(start Position, first byte) Token {
	var b strings.Builder
	var isEscape bool
	for l.HasBytes() {
		c := l.Next()
		if c == '\\' {
			isEscape = !isEscape
		} else if c == first && !isEscape {
			tok := newToken(start, String, b.String())
			tok.Attributes = StringAttrs{QuoteStyle: first}
			return tok
		}
		b.WriteByte(c)
	}
	tok := newToken(start, String, b.String())
	tok.Attributes = StringAttrs{
		QuoteStyle:   first,
		Unterminated: true,
	}
	return tok
}

func (l *lexer) ParseNumber(start Position, first byte) Token {
	var b strings.Builder
	var hadDecimal bool
	b.WriteByte(first)
	for l.HasBytes() {
		c := l.Current()
		if (c >= '0' && c <= '9') || c == '_' || (c == '.' && !hadDecimal) {
			b.WriteByte(c)
			l.Next()
		} else {
			break
		}
	}
	return newToken(start, Numeric, b.String())
}

func (l *lexer) ParseUnquoted(start Position, first byte) Token {
	var b strings.Builder
	b.WriteByte(first)
	for l.HasBytes() {
		c := l.Current()
		if c == '\n' {
			break
		}
		l.Next()
		b.WriteByte(c)
	}
	tok := newToken(start, Identifier, b.String())
	tok.Attributes = StringAttrs{Unquoted: true}
	return tok
}

func (l *lexer) ParseIdent(start Position, first byte) Token {
	var b strings.Builder
	b.WriteByte(first)
	for l.HasBytes() {
		c := rune(l.Current())
		if unicode.IsLetter(c) || unicode.IsDigit(c) || allowedInVar[c] {
			b.WriteRune(c)
			l.Next()
		} else {
			break
		}
	}
	return newToken(start, Identifier, b.String())
}

func (l *lexer) ParseComment(start Position, isBlock bool) Token {
	var b strings.Builder
	cmtLevel := 1
	for l.HasBytes() {
		c := l.Current()
		switch {
		case isBlock && c == '/' && l.Next() == '*':
			cmtLevel++
			l.Next()
			l.Next()
			b.WriteString("/*")
			continue
		case isBlock && c == '*' && l.Next() == '/':
			cmtLevel--
			l.Next()
			l.Next()
			if cmtLevel == 0 {
				tok := newToken(start, Identifier, b.String())
				tok.Attributes = CommentAttrs{Block: true}
				return tok
			}
			b.WriteString("*/")
			continue
		case !isBlock && c == '\n':
			l.Next()
			b.WriteByte('\n')
			tok := newToken(start, Identifier, b.String())
			tok.Attributes = CommentAttrs{Block: false}
			return tok
		}
		b.WriteByte(c)
		l.Next()
	}
	tok := newToken(start, Identifier, b.String())
	tok.Attributes = CommentAttrs{
		Block:        isBlock,
		Unterminated: isBlock,
	}
	return tok
}
