package reader

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/pkg/klarml/internal/errors"
)

func (rd *Reader) readRune() (rune, error) {
	if rd.needsMore() {
		if err := rd.refill(); err != nil {
			if err == io.EOF {
				return 0, io.EOF
			}
			panic(ReadError{err})
		}
	}
	r, n := utf8.DecodeRune(rd.buffer[rd.pos:])
	rd.pos += n
	if r == '\n' {
		rd.offset.Line++
		rd.offset.Col = 1
	} else {
		rd.offset.Col++
	}
	return r, nil
}

func (rd *Reader) peekRune() (rune, int, error) {
	if rd.needsMore() {
		if err := rd.refill(); err != nil {
			if err == io.EOF {
				return 0, 0, io.EOF
			}
			panic(ReadError{err})
		}
	}
	r, n := utf8.DecodeRune(rd.buffer[rd.pos:])
	return r, n, nil
}

func (rd *Reader) currRune() (rune, int, error) {
	if err := rd.tryRefill(); err != nil {
		return 0, 0, err
	}
	r, n := utf8.DecodeRune(rd.buffer[rd.pos:])
	return r, n, nil
}

func (rd *Reader) advanceBytes(n int) {
	rd.pos += n
	rd.offset.Col += 1
}

func (rd *Reader) readToken() Token {
	for {
		start := rd.offset
		r, err := rd.readRune()
		if err == io.EOF {
			return Token{Kind: EOF}
		}
		switch r {
		case ' ', '\t':
			continue
		case '\n':
			return Token{Kind: Newline, Pos: start, Src: string(r)}
		case '-':
			if curr, _, _ := rd.currRune(); lexer.IsDigit(curr) {
				return rd.readNumber(r, start)
			}
			return Token{Kind: Dash, Pos: start, Src: string(r)}
		case '+', '.':
			if curr, _, _ := rd.currRune(); lexer.IsDigit(curr) {
				return rd.readNumber(r, start)
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return rd.readNumber(r, start)
		case '[':
			return Token{Kind: LeftBracket, Pos: start, Src: string(r)}
		case ']':
			return Token{Kind: RightBracket, Pos: start, Src: string(r)}
		case '@':
			return Token{Kind: At, Pos: start, Src: string(r)}
		case '$':
			return Token{Kind: Dollar, Pos: start, Src: string(r)}
		case ':':
			return Token{Kind: Colon, Pos: start, Src: string(r)}
		case '{':
			return Token{Kind: LeftCurly, Pos: start, Src: string(r)}
		case '}':
			return Token{Kind: RightCurly, Pos: start, Src: string(r)}
		case ',':
			return Token{Kind: Comma, Pos: start, Src: string(r)}
		case '<':
			if curr, n, _ := rd.currRune(); curr == '-' {
				rd.advanceBytes(n)
				return Token{Kind: Arrow, Pos: start, Src: string(r)}
			}
		case '>':
			if curr, n, _ := rd.currRune(); curr == '"' || curr == '\'' {
				rd.advanceBytes(n)
				return rd.readQuotedString(curr, start, true)
			}
		case utf8.RuneError:
			return Token{Kind: Illegal, Pos: start, Src: string(r)}
		case '"', '\'':
			return rd.readQuotedString(r, start, false)
		default:
			switch {
			case unicode.IsSpace(r):
				continue
			case unicode.IsLetter(r):

			}
		}
		return rd.readUnquotedString(nil, start)
	}
}

func (rd *Reader) readQuotedString(quote rune, start lexer.Position, wrap bool) Token {
	var b strings.Builder
	ret := func(err errors.ErrorCode) Token {
		return Token{
			Kind:  String,
			Src:   b.String(),
			Pos:   start,
			Attrs: attrs{"err": err, "quote": quote, "wrap": wrap},
		}
	}
	if wrap {
		b.WriteRune('>')
	}
	b.WriteRune(quote)
	for {
		c, err := rd.readRune()
		if err == io.EOF {
			return ret(ErrUnterminatedString)
		}
		b.WriteRune(c)
		if c == quote {
			return ret(0)
		}
	}
}

func (rd *Reader) readNumber(first rune, start lexer.Position) Token {
	var b strings.Builder
	isNumber := true
	var isDecimal, wasUnderscore bool
	value := func() Token {
		tok := Token{Kind: Number, Src: b.String(), Pos: start}
		if !isNumber || len(tok.Src) == 1 {
			tok.Kind = String
		}
		return tok
	}
	// Check first digit or +, -, .
	b.WriteRune(first)
	for {
		c, size, err := rd.currRune()
		if err != nil {
			return value()
		}
		switch {
		case c == '_' && wasUnderscore, c == '.' && isDecimal:
			isNumber = false
		case c == '_':
			wasUnderscore = true
		case c == '.':
			isDecimal = true
		case unicode.IsSpace(c), rd.isPunct(c):
			return value()
		case !lexer.IsDigit(c):
			isNumber = false
		}
		b.WriteRune(c)
		rd.advanceBytes(size)
		if !isNumber {
			return rd.readUnquotedString(&b, start)
		}
	}
	// return value(), nil
}

func (rd *Reader) isPunct(r rune) bool {
	switch r {
	case '\n', '@', '$', ']', '}':
		return true
	case ',':
		return rd.comma
	}
	return false
}

func (rd *Reader) readUnquotedString(b *strings.Builder, start lexer.Position) Token {
	if b == nil {
		b = &strings.Builder{}
	}
	for {
		c, n, err := rd.currRune()
		if err != nil || rd.isPunct(c) {
			break
		}
		rd.advanceBytes(n)
		b.WriteRune(c)
	}
	str := b.String()
	if str == "true" || str == "false" {
		return Token{
			Kind:  Boolean,
			Src:   str,
			Pos:   start,
			Attrs: attrs{"value": str == "true"},
		}
	}
	return Token{Kind: String, Src: str, Pos: start}
}
