package klon

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
)

const (
	noComma     uint8 = 1 << iota // Disallow commas in unquoted strings
	objectValue                   // Allow more characters in unquoted strings
	allowDot                      // Allow dots in unquoted strings
	key                           // Currently reading a key
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

func (rd *reader) AdvanceRune() (rune, error) { return rd.readRune() }
func (rd *reader) CurrRune() (rune, error) {
	if rd.pos >= len(rd.buffer) {
		if err := rd.refill(); err != nil {
			return 0, err
		}
	}
	r, _ := utf8.DecodeRune(rd.buffer[rd.pos:])
	return r, nil
}

func (rd *reader) PeekRune() (rune, error) {
	if rd.pos >= len(rd.buffer) {
		if err := rd.refill(); err != nil {
			return 0, err
		}
	}
	_, n := utf8.DecodeRune(rd.buffer[rd.pos:])
	nextPos := rd.pos + n
	if nextPos >= len(rd.buffer) {
		if rd.reader != nil {
			if err := rd.refill(); err != nil {
				return 0, err
			}
			_, n = utf8.DecodeRune(rd.buffer[rd.pos:])
			nextPos = rd.pos + n
		}
	}
	if nextPos >= len(rd.buffer) {
		return 0, io.EOF
	}
	r2, _ := utf8.DecodeRune(rd.buffer[nextPos:])
	return r2, nil
}
func (rd *reader) Position() lexer.Position { return rd.offset }

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

func (rd *reader) utf8Error(back int) {
	pos := rd.offset
	if back > 0 {
		pos = pos.Sub(0, uint32(back))
	}
	rd.rangeError(
		klonerrs.ErrIllegalCharacter, ranges.SingleChar(pos),
		"Invalid Unicode character",
	)
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
			if (rd.parseFlags & objectValue) == 0 {
				return Token{Kind: Dash, Pos: start, Src: string(r), BufPos: bufPos}
			}
		case '+':
			if curr, _, _ := rd.currRune(); curr >= '0' && curr <= '9' {
				tok := rd.readNumber(r, start, bufPos)
				rd.tokenError(klonerrs.ErrLeadingPlusSign, tok, "Redundant positive number prefix")
				return tok
			}
		case '.':
			if (rd.parseFlags & allowDot) == 0 {
				return Token{Kind: Dot, Pos: start, Src: ".", BufPos: bufPos}
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return rd.readNumber(r, start, bufPos)
		case '[':
			return Token{Kind: LeftBracket, Pos: start, Src: string(r), BufPos: bufPos}
		case ']':
			return Token{Kind: RightBracket, Pos: start, Src: string(r), BufPos: bufPos}
		case '@':
			if curr, _, _ := rd.currRune(); !unicode.IsSpace(curr) {
				return rd.readClass(start, bufPos)
			}
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
				return rd.readQuotedString(curr, true, start, bufPos)
			}
		case utf8.RuneError:
			tok := Token{Kind: Illegal, Pos: start, Src: string(r), BufPos: bufPos}
			rd.utf8Error(1)
			return tok
		case '"', '\'':
			return rd.readQuotedString(r, false, start, bufPos)
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

func (rd *reader) readQuotedString(quote rune, wrap bool,
	start lexer.Position, bufPos int,
) Token {
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
		switch {
		case escape:
			escape = false
		case r == quote:
			return ret(false)
		case r == utf8.RuneError:
			// No invalid UTF-8
			rd.utf8Error(1)
		case r == '\\':
			escape = true
		}
	}
}

func (rd *reader) readNumber(first rune, start lexer.Position, bufPos int) Token {
	var prefix string
	isNumber := true
	// Leading -/+ sign ('+' is invalid, but checked outside readNumber)
	if first == '-' || first == '+' {
		prefix = string(first)
		r, n, err := rd.currRune()
		if err != nil || !lexer.IsDigit(r) {
			b := &strings.Builder{}
			b.WriteRune(first)
			return rd.readUnquotedString(b, start, bufPos)
		}
		rd.advanceBytes(n)
		first = r
	}
	// Use the same number format as Klar
	literal, params := lexer.ReadNumber(rd, first)
	// Read a string if the literal ends in an invalid underscore
	if literal[len(literal)-1] == '_' {
		isNumber = false
	}
	// Check if we should read an unquoted string instead
	r, _, err := rd.currRune()
	isDelim := err != nil || unicode.IsSpace(r) || rd.isPunct(r) || r == ','
	if !isNumber || !isDelim {
		// Read an unquoted string instead
		b := &strings.Builder{}
		b.WriteString(prefix + literal)
		return rd.readUnquotedString(b, start, bufPos)
	}

	return Token{
		Kind:   Number,
		Src:    prefix + literal,
		Pos:    start,
		BufPos: bufPos,
		Attrs:  attrs{"params": params},
	}
}

func (rd *reader) isPunct(r rune) bool {
	switch r {
	case '\n', '@', '$', '[', ']', '{', '}':
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
		// Error on invalid UTF-8
		if r == utf8.RuneError {
			rd.utf8Error(0)
		}
		rd.advanceBytes(n)
		b.WriteRune(r)
	}
	str := strings.TrimSpace(b.String()) // Trim whitespace around
	switch str {
	case "true", "false":
		return Token{
			Kind:   Boolean,
			Src:    str,
			Pos:    start,
			BufPos: bufPos,
			Attrs:  attrs{"value": str == "true"},
		}
	case "none":
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
			Attrs:  attrs{"err": klonerrs.ErrUnterminatedVar},
		}
	}
	var (
		isBrace = r == '{'
		unterm  = true
		tokErr  klonerrs.Code
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
		case isBrace && b.Len() == 2 && unicode.IsDigit(r),
			!isBrace && b.Len() == 1 && unicode.IsDigit(r):
			tokErr = klonerrs.ErrInvalidIdentifier
		case !isValidIdentChar(r):
			break loop
		}
		rd.advanceBytes(n)
		b.WriteRune(r)
	}
	if isBrace && unterm {
		tokErr = klonerrs.ErrUnterminatedVar
	}
	return Token{
		Kind:   Variable,
		Src:    b.String(),
		Pos:    start,
		BufPos: bufPos,
		Attrs:  attrs{"err": tokErr, "brace": isBrace},
	}
}

func (rd *reader) readClass(start lexer.Position, bufPos int) Token {
	var b strings.Builder
	var invalid bool
	b.WriteString("@")
loop:
	for {
		r, size, err := rd.currRune()
		switch {
		case err != nil:
			break loop
		case b.Len() == 1 && unicode.IsDigit(r):
			invalid = true
		case !isValidIdentChar(r):
			break loop
		}
		b.WriteRune(r)
		rd.advanceBytes(size)
	}
	return Token{
		Kind:   AtRef,
		Src:    b.String(),
		Pos:    start,
		BufPos: bufPos,
		Attrs:  attrs{"invalid": invalid, "end": rd.offset},
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
		BaseNode: ast.BaseNode{ranges.Range{start, rd.offset}},
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
		rd.rangeError(klonerrs.ErrUnterminatedComment, cmt.Range, "Expected '*/' to end block comment")
	}
	rd.comments = append(rd.comments, cmt)
}
