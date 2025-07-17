package parser

import (
	"slices"
	"strings"
	"unicode"
)

type lexer struct {
	Bytes []byte
	Index int
	Position
}

func (l *lexer) Shift() byte {
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

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func (l *lexer) Peek() byte     { return l.Bytes[l.Index+1] }
func (l *lexer) Current() byte  { return l.Bytes[l.Index] }
func (l *lexer) HasBytes() bool { return l.Index < len(l.Bytes) }

var wasColon bool

func (l *lexer) Tokenize() Token {
	var (
		_colon bool
		pos    = l.Position
		b      = l.Shift()
	)
	defer func() {
		if !_colon {
			wasColon = false
		}
	}()
	switch b {
	case ':':
		_colon = true
		return newToken(pos, Colon, ":")
	case '-':
		if next := l.Peek(); isDigit(next) || next == '.' {
			break
		}
		return newToken(pos, Hyphen, "-")
	case '{':
		return newToken(pos, LeftBrace, "{")
	case '}':
		return newToken(pos, RightBrace, "}")
	case '.':
		return newToken(pos, Period, ".")
	case ',':
		return newToken(pos, Comma, ",")
	case '$':
		return newToken(pos, Dollar, "$")
	case '\n':
		return newToken(pos, Newline, "\n")
	case '@':
		return l.ParseNamespace(pos)
	case '/':
		next := l.Peek()
		if next == '/' || next == '*' {
			l.Shift()
			return l.ParseComment(pos, next == '*')
		}
	case '"', '\'':
		return l.ParseString(pos, b)
	}
	r := rune(b)
	switch {
	case unicode.IsLetter(r) && !wasColon:
		return l.ParseIdent(pos, b)
	case isDigit(b), r == '_', r == '+', r == '.', r == '-':
		return l.ParseIdent(pos, b)
	case unicode.IsSpace(r):
		return l.Tokenize()
	default:
		return l.ParseUnquoted(pos, b)
	}
}

var allowedInVar = map[rune]bool{
	'.': true, '+': true, '-': true, '_': true, '\\': true,
}

func (l *lexer) ParseNamespace(start Position) Token {
	ident := l.ParseIdent(l.Position, '@').Source
	return newToken(start, TokenNamespace, ident)
}

func (l *lexer) ParseString(start Position, first byte) Token {
	var b strings.Builder
	var isEscape bool
	for l.HasBytes() {
		c := l.Shift()
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
	var hadDecimal, hadUnderscore bool
	b.WriteByte(first)
	for l.HasBytes() {
		c := l.Current()
		if c == '_' && !hadUnderscore {
			hadUnderscore = true
		} else if c == '.' && !hadDecimal && !hadUnderscore {
			hadDecimal = true
		} else if isDigit(c) {
		} else {
			break
		}
		hadUnderscore = false
		b.WriteByte(c)
		l.Shift()
	}
	return newToken(start, Numeric, b.String())
}

func (l *lexer) ParseUnquoted(start Position, first byte) Token {
	var b strings.Builder
	var isEscape bool
	b.WriteByte(first)
loop:
	for l.HasBytes() {
		c := l.Current()
		switch c {
		case '\n':
			break loop
		case '\\':
			isEscape = !isEscape
		case '@', ',':
			if !isEscape {
				break loop
			}
		}
		l.Shift()
		b.WriteByte(c)
	}
	tok := newToken(start, Identifier, b.String())
	tok.Attributes = StringAttrs{Unquoted: true}
	return tok
}

func (l *lexer) ParseIdent(start Position, first byte) Token {
	var b strings.Builder
	var isEscape bool
	b.WriteByte(first)
	for l.HasBytes() {
		currByte := l.Current()
		c := rune(currByte)
		switch {
		case c == '\\':
			isEscape = true
			fallthrough
		case isEscape, unicode.IsLetter(c), unicode.IsDigit(c), allowedInVar[c]:
			b.WriteByte(currByte)
			l.Shift()
			isEscape = false
			continue
		}
		break
	}
	return newToken(start, Identifier, b.String())
}

func (l *lexer) ParseComment(start Position, isBlock bool) Token {
	var b strings.Builder
	cmtLevel := 1
	for l.HasBytes() {
		c := l.Current()
		switch {
		case isBlock && c == '/' && l.Shift() == '*':
			cmtLevel++
			l.Shift()
			l.Shift()
			b.WriteString("/*")
			continue
		case isBlock && c == '*' && l.Shift() == '/':
			cmtLevel--
			l.Shift()
			l.Shift()
			if cmtLevel == 0 {
				tok := newToken(start, Identifier, b.String())
				tok.Attributes = CommentAttrs{Block: true}
				return tok
			}
			b.WriteString("*/")
			continue
		case !isBlock && c == '\n':
			l.Shift()
			b.WriteByte('\n')
			tok := newToken(start, Identifier, b.String())
			tok.Attributes = CommentAttrs{Block: false}
			return tok
		}
		b.WriteByte(c)
		l.Shift()
	}
	tok := newToken(start, Identifier, b.String())
	tok.Attributes = CommentAttrs{
		Block:        isBlock,
		Unterminated: isBlock,
	}
	return tok
}

func Tokenize(bytes []byte) []Token {
	tokens := make([]Token, 0, len(bytes)/2)
	l := lexer{
		Bytes:    bytes,
		Index:    0,
		Position: Position{1, 1},
	}
	for l.HasBytes() {
		tokens = append(tokens, l.Tokenize())
	}
	tokens = append(tokens, newToken(l.Position, EOF, ""))
	tokens = slices.Clip(tokens)
	return tokens
}