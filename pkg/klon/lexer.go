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
	allowDot                      // Allow dots in unquoted strings
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
		rd.resetLine()
	} else {
		rd.offset.Col++
	}
	return r, nil
}

func (rd *reader) resetLine() {
	rd.offset.Line++
	rd.offset.Col = 1
}

func (rd *reader) resetLineIf(r rune) {
	if r == '\n' {
		rd.resetLine()
	}
}

func (rd *reader) peekRune() (rune, int, error) {
	if rd.needsMore() {
		if err := rd.tryRefill(); err != nil {
			return 0, 0, err
		}
	}
	if rd.pos+1 >= len(rd.buffer) {
		return 0, 0, io.EOF
	}
	r, n := utf8.DecodeRune(rd.buffer[rd.pos+1:])
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
		bufPos := rd.pos
		r, err := rd.readRune()
		if err == io.EOF {
			return Token{Kind: EOF, Pos: start, BufPos: bufPos}
		}
		switch r {
		case ' ', '\t':
			continue
		case '\n':
			return Token{Kind: Newline, Pos: start, Src: string(r), BufPos: bufPos}
		case '/':
			if curr, n, _ := rd.currRune(); curr == '/' {
				rd.advanceBytes(n)
				rd.readLineComment(start)
				continue
			} else if curr == '*' {
				rd.advanceBytes(n)
				rd.readBlockComment(start)
				continue
			}
		case '-':
			if curr, _, _ := rd.currRune(); curr >= '0' && curr <= '9' {
				return rd.readNumber(r, start, bufPos)
			}
			return Token{Kind: Dash, Pos: start, Src: string(r), BufPos: bufPos}
		case '+':
			if curr, _, _ := rd.currRune(); curr >= '0' && curr <= '9' {
				return rd.readNumber(r, start, bufPos)
			}
		case '.':
			// TODO: Don't allow leading/trailing decimal point for numbers
			if curr, _, _ := rd.currRune(); curr >= '0' && curr <= '9' {
				return rd.readNumber(r, start, bufPos)
			}
			if (rd.parseFlags & allowDot) == 0 {
				return Token{Kind: String, Pos: start, Src: ".", BufPos: bufPos}
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return rd.readNumber(r, start, bufPos)
		case '[':
			return Token{Kind: LeftBracket, Pos: start, Src: string(r), BufPos: bufPos}
		case ']':
			return Token{Kind: RightBracket, Pos: start, Src: string(r), BufPos: bufPos}
		case '@':
			return Token{Kind: At, Pos: start, Src: string(r), BufPos: bufPos}
		case '$':
			return rd.readVariable(start, bufPos)
		case ':':
			if (rd.parseFlags & objectValue) == 0 {
				return Token{Kind: Colon, Pos: start, Src: string(r), BufPos: bufPos}
			}
		case '{':
			return Token{Kind: LeftCurly, Pos: start, Src: string(r), BufPos: bufPos}
		case '}':
			return Token{Kind: RightCurly, Pos: start, Src: string(r), BufPos: bufPos}
		case ',':
			if (rd.parseFlags & noComma) != 0 {
				return Token{Kind: Comma, Pos: start, Src: string(r), BufPos: bufPos}
			}
		case '<':
			if curr, n, _ := rd.currRune(); curr == '-' {
				rd.advanceBytes(n)
				return Token{Kind: Arrow, Pos: start, Src: string(r), BufPos: bufPos}
			}
		case '>':
			if curr, n, _ := rd.currRune(); curr == '"' || curr == '\'' {
				rd.advanceBytes(n)
				return rd.readQuotedString(curr, start, bufPos, true)
			}
		case utf8.RuneError:
			tok := Token{Kind: Illegal, Pos: start, Src: string(r), BufPos: bufPos}
			rd.tokenError(ErrIllegalCharacter, tok, "Invalid Unicode character")
			return tok
		case '"', '\'':
			return rd.readQuotedString(r, start, bufPos, false)
		default:
			if unicode.IsSpace(r) {
				continue
			}
		}
		b := &strings.Builder{}
		b.WriteRune(r)
		return rd.readUnquotedString(b, start, bufPos)
	}
}

func (rd *reader) readQuotedString(quote rune, start lexer.Position, bufPos int, wrap bool) Token {
	var b strings.Builder
	ret := func(unterm bool) Token {
		return Token{
			Kind:   String,
			Src:    b.String(),
			Pos:    start,
			BufPos: bufPos,
			Attrs:  attrs{"unterm": unterm, "quote": quote, "wrap": wrap},
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

func (rd *reader) readNumber(first rune, start lexer.Position, bufPos int) Token {
	var b strings.Builder
	isNumber := true
	var isDecimal, wasUnderscore bool
	value := func() Token {
		tok := Token{Kind: Number, Src: b.String(), Pos: start, BufPos: bufPos}
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
			return rd.readUnquotedString(&b, start, bufPos)
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
	case '.':
		return (rd.parseFlags & allowDot) == 0
	}
	return false
}

func (rd *reader) readUnquotedString(b *strings.Builder, start lexer.Position, bufPos int) Token {
	for {
		r, n, err := rd.currRune()
		if err != nil || rd.isPunct(r) {
			break
		}
		// Comment between or after string
		if r == '/' {
			if r2, _, _ := rd.peekRune(); r2 == '/' || r2 == '*' {
				break
			}
		}
		rd.advanceBytes(n)
		b.WriteRune(r)
	}
	str := strings.TrimSpace(b.String())
	switch str {
	case "true", "false":
		return Token{
			Kind:   Boolean,
			Src:    str,
			Pos:    start,
			BufPos: bufPos,
			Attrs:  attrs{"value": str == "true"},
		}
	case "null":
		return Token{Kind: None, Src: str, Pos: start, BufPos: bufPos}
	}
	return Token{
		Kind:   String,
		Src:    str,
		Pos:    start,
		BufPos: bufPos,
		Attrs:  attrs{"end": rd.offset, "quote": rune(0)},
	}
}

func (rd *reader) readVariable(start lexer.Position, bufPos int) Token {
	var b strings.Builder
	b.WriteByte('$')
	r, n, err := rd.currRune()
	if err != nil {
		return Token{
			Kind:   Variable,
			Src:    b.String(),
			Pos:    start,
			BufPos: bufPos,
			Attrs:  attrs{"err": ErrUnterminatedVar},
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
		Kind:   Variable,
		Src:    b.String(),
		Pos:    start,
		BufPos: bufPos,
		Attrs:  attrs{"err": tokErr, "brace": isBrace},
	}
}

func isValidIdentChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (rd *reader) readLineComment(start lexer.Position) {
	var b strings.Builder
	b.WriteString("//")
	for {
		r, size, err := rd.currRune()
		if err != nil || r == '\n' {
			break
		}
		b.WriteRune(r)
		rd.advanceBytes(size)
	}
	rd.comments = append(rd.comments, &ast.Comment{
		BaseNode: ast.BaseNode{ranges.Range{Start: start, End: rd.offset}},
		Source:   b.String(),
	})
}

func (rd *reader) readBlockComment(start lexer.Position) {
	var b strings.Builder
	b.WriteString("/*")
	depth := 1
	for {
		r, err := rd.readRune()
		if err != nil {
			break
		}
		b.WriteRune(r)
		if r == '*' {
			if r2, n, _ := rd.currRune(); r2 == '/' {
				b.WriteRune(r2)
				rd.advanceBytes(n)
				if depth--; depth == 0 {
					break
				}
			}
		} else if r == '/' {
			if r2, n, _ := rd.currRune(); r2 == '*' {
				b.WriteRune(r2)
				rd.advanceBytes(n)
				depth++
			}
		}
	}
	cmt := &ast.Comment{
		BaseNode: ast.BaseNode{ranges.Range{start, rd.offset}},
		Block:    true,
		Source:   b.String(),
	}
	if depth > 0 {
		// Unterminated block comment
		rd.rangeError(ErrUnterminatedComment, cmt.Range, "Expected '*/' to end block comment")
	}
	rd.comments = append(rd.comments, cmt)
}
