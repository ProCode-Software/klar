package klon

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

const (
	noComma     uint8 = 1 << iota // Disallow commas in unquoted strings
	objectValue                   // Allow more characters in unquoted strings
)

func (rd *reader) readRune() (rune, error) {
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

func (rd *reader) peekRune() (rune, int, error) {
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

func (rd *reader) currRune() (rune, int, error) {
	if err := rd.tryRefill(); err != nil {
		return 0, 0, err
	}
	r, n := utf8.DecodeRune(rd.buffer[rd.pos:])
	return r, n, nil
}

func (rd *reader) advanceBytes(n int) {
	rd.pos += n
	rd.offset.Col += 1
}

func (rd *reader) readToken() Token {
	for {
		start := rd.offset
		r, err := rd.readRune()
		if err == io.EOF {
			return Token{Kind: EOF, Pos: start}
		}
		switch r {
		case ' ', '\t':
			continue
		case '\n':
			return Token{Kind: Newline, Pos: start, Src: string(r)}
		case '/':
			// TODO
		case '-':
			if curr, _, _ := rd.currRune(); curr >= '0' && curr <= '9' {
				return rd.readNumber(r, start)
			}
			return Token{Kind: Dash, Pos: start, Src: string(r)}
		case '+', '.':
			if curr, _, _ := rd.currRune(); curr >= '0' && curr <= '9' {
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
			return rd.readVariable(start)
		case ':':
			if (rd.parseFlags & objectValue) == 0 {
				return Token{Kind: Colon, Pos: start, Src: string(r)}
			}
		case '{':
			return Token{Kind: LeftCurly, Pos: start, Src: string(r)}
		case '}':
			return Token{Kind: RightCurly, Pos: start, Src: string(r)}
		case ',':
			if (rd.parseFlags & noComma) != 0 {
				return Token{Kind: Comma, Pos: start, Src: string(r)}
			}
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
			tok := Token{Kind: Illegal, Pos: start, Src: string(r)}
			rd.tokenError(ErrIllegalCharacter, tok, "Invalid Unicode character")
			return tok
		case '"', '\'':
			return rd.readQuotedString(r, start, false)
		default:
			if unicode.IsSpace(r) {
				continue
			}
		}
		b := &strings.Builder{}
		b.WriteRune(r)
		return rd.readUnquotedString(b, start)
	}
}

func (rd *reader) readQuotedString(quote rune, start lexer.Position, wrap bool) Token {
	var b strings.Builder
	ret := func(unterm bool) Token {
		return Token{
			Kind:  String,
			Src:   b.String(),
			Pos:   start,
			Attrs: attrs{"unterm": unterm, "quote": quote, "wrap": wrap},
		}
	}
	if wrap {
		b.WriteRune('>')
	}
	b.WriteRune(quote)
	var escape bool
	for {
		r, err := rd.readRune()
		if err == io.EOF {
			return ret(true)
		}
		b.WriteRune(r)
		if !escape {
			if r == quote {
				return ret(false)
			} else if r == '\\' {
				escape = true
				continue
			}
		}
		escape = false
	}
}

func (rd *reader) readNumber(first rune, start lexer.Position) Token {
	var b strings.Builder
	isNumber := true
	var isDecimal, wasUnderscore bool
	value := func() Token {
		tok := Token{Kind: Number, Src: b.String(), Pos: start}
		if !isNumber || tok.Src[0] < '0' || tok.Src[0] > '9' {
			tok.Kind = String
		}
		return tok
	}
	// Check first digit or +, -, .
	b.WriteRune(first)
	for {
		r, size, err := rd.currRune()
		if err != nil {
			return value()
		}
		switch {
		case r == '_' && wasUnderscore, r == '.' && isDecimal:
			isNumber = false
		case r == '_':
			wasUnderscore = true
		case r == '.':
			isDecimal = true
		case unicode.IsSpace(r), rd.isPunct(r):
			return value()
		case r < '0' || r > '9':
			isNumber = false
		}
		b.WriteRune(r)
		rd.advanceBytes(size)
		if !isNumber {
			return rd.readUnquotedString(&b, start)
		}
	}
}

func (rd *reader) isPunct(r rune) bool {
	switch r {
	case '\n', '@', '$', ']', '}':
		return true
	case ':':
		return (rd.parseFlags & objectValue) == 0
	case ',':
		return (rd.parseFlags & noComma) != 0
	}
	return false
}

func (rd *reader) readUnquotedString(b *strings.Builder, start lexer.Position) Token {
	for {
		r, n, err := rd.currRune()
		if err != nil || rd.isPunct(r) {
			break
		}
		rd.advanceBytes(n)
		b.WriteRune(r)
	}
	str := strings.TrimSpace(b.String())
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

func (rd *reader) readVariable(start lexer.Position) Token {
	var b strings.Builder
	b.WriteByte('$')
	r, n, err := rd.currRune()
	if err != nil {
		return Token{
			Kind:  Variable,
			Src:   b.String(),
			Pos:   start,
			Attrs: attrs{"err": ErrUnterminatedVar},
		}
	}
	var (
		isBrace = r == '{'
		unterm  = true
		tokErr  Code
	)
	if isBrace {
		rd.advanceBytes(n)
		b.WriteRune(r)
	}
loop:
	for {
		r, n, err = rd.currRune()
		switch {
		case err != nil:
			break loop
		case r == '}' && isBrace:
			unterm = false
			b.WriteRune(r)
			rd.advanceBytes(n)
			break loop
		case b.Len() == 1 && unicode.IsDigit(r):
			tokErr = ErrInvalidIdentifier
		case !isValidIdentChar(r):
			break loop
		}
		rd.advanceBytes(n)
		b.WriteRune(r)
	}
	if isBrace && unterm {
		tokErr = ErrUnterminatedVar
	}
	return Token{
		Kind:  Variable,
		Src:   b.String(),
		Pos:   start,
		Attrs: attrs{"err": tokErr, "brace": isBrace},
	}
}

func isValidIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
