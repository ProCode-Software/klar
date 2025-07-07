package lexer

import (
	"unicode"
)

type EscapeType int

const (
	_ EscapeType = iota
	EscCharacter
	EscHex
	EscUnicode
	EscInterpolation
)

type (
	EscapeError int
	EscapeMap   = map[Position]StringEscape
)

const (
	_ EscapeError = iota
	ErrEscapeTooShort
	ErrEscapeTooLong
	ErrEscapeUnterm
	ErrEscapeExpHex
	ErrEscapeUnknown
)

type StringEscape struct {
	Type          EscapeType
	Value         string
	Interpolated  []Token
	Invalid       EscapeError
	ErrorPosition Position
}

func isHex(r rune) bool {
	return unicode.Is(unicode.ASCII_Hex_Digit, r)
}

func (l *Lexer) parseUnicodeEsc(delim rune) StringEscape {
	var (
		err    EscapeError
		errPos Position
		last   rune
	)
	invalid := func(reason EscapeError) {
		err, errPos = reason, l.Pos
	}
	esc := l.TokenizeEOFFunc(func(r rune, b *Builder) bool {
		l := b.Len()
		switch {
		case r == '}', isHex(r), l == 0 && r == '{':
			b.WriteRune(r)
		case l > 0 && last == '}':
			return false
		case l > 6 && r != '}':
			invalid(ErrEscapeTooLong)
			return false
		case l < 2 && r == '}':
			invalid(ErrEscapeTooShort)
			b.WriteRune(r)
		case r == delim:
			invalid(ErrEscapeUnterm)
			return false
		default:
			invalid(ErrEscapeExpHex)
			return false
		}
		last = r
		return true
	}, func() {
		if last != '}' {
			invalid(ErrEscapeUnterm)
		}
	})

	return StringEscape{
		Type:          EscUnicode,
		Value:         `\u` + esc,
		Invalid:       err,
		ErrorPosition: errPos,
	}
}

func (l *Lexer) parseHexEsc(delim rune) StringEscape {
	var (
		err     EscapeError
		errPos  Position
		unterm  bool
		invalid = func() { err, errPos = ErrEscapeExpHex, l.Pos }
	)
	esc := l.TokenizeEOFFunc(func(r rune, b *Builder) bool {
		l := b.Len()
		switch {
		case l == 2:
			return false
		case isHex(r):
			b.WriteRune(r)
		case r == delim:
			invalid()
			return false
		default:
			invalid()
			b.WriteRune(r)
		}
		return true
	}, func() { unterm = true })

	if unterm && len(esc) < 2 {
		invalid()
	}

	return StringEscape{
		Type:          EscHex,
		Value:         `\x` + esc,
		Invalid:       err,
		ErrorPosition: errPos,
	}
}

// '{' already parsed
func (l *Lexer) parseStrInterp() StringEscape {
	var (
		err     EscapeError
		errPos  Position
		tokens  []Token
		braceCt = 1
		b       Builder
	)
loop:
	for {
		t := l.Tokenize()
		switch t.Kind {
		case EOF:
			err, errPos = ErrEscapeUnterm, l.Pos
			break loop
		case LeftCurlyBrace, HashLeftCurlyBrace:
			braceCt++
		case RightCurlyBrace:
			braceCt--
			if braceCt == 0 {
				if len(tokens) == 0 {
					err, errPos = ErrEscapeTooShort, l.Pos
				} else {
					tokens[len(tokens)-1].SetAttribute("end", l.Pos)
				}
				b.WriteString(t.Source)
				break loop
			}
		}
		tokens = append(tokens, *t)
		b.WriteString(t.Source)
	}
	return StringEscape{
		Type:          EscInterpolation,
		Interpolated:  tokens,
		Value:         "{" + b.String(),
		Invalid:       err,
		ErrorPosition: errPos,
	}
}

/*
Beginning is already parsed

There are three types of strings in Klar:

	Double-quote "": Interpolation, escapes
	Single-quote '': Raw, no escapes
	Backtick ``: Interpolation, escapes, multiline
*/
func (l *Lexer) ParseString(pos Position, delim rune) *Token {
	var (
		esc                         string
		b                           Builder
		isEscape, isNewline, unterm bool
		escapes                     = make(EscapeMap)
		escStart, end               Position
	)
	escape := func(typ EscapeType, err EscapeError) {
		e := StringEscape{Type: typ, Value: `\` + esc, Invalid: err}
		if err > 0 {
			e.ErrorPosition = l.Pos
		}
		escapes[escStart], isEscape, esc = e, false, ""
	}
	parsedEscape := func(e StringEscape, p rune) {
		b.WriteString(string(p) + e.Value[2:])
		escapes[escStart], isEscape, esc = e, false, ""
	}
loop:
	for {
		r, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			unterm = true
			break loop
		}
		l.Pos.Col++
		if isEscape {
			esc += string(r)
		}
		switch r {
		case delim:
			if isEscape {
				escape(EscCharacter, 0)
			} else {
				b.WriteRune(r)
				end = l.Pos
				break loop
			}
		case '\\':
			if isEscape {
				escape(EscCharacter, 0)
			} else if delim != '\'' {
				isEscape, escStart = true, l.prevCol()
			}
		case '{':
			if isEscape {
				escape(EscCharacter, 0)
			} else if delim != '\'' {
				e := l.parseStrInterp()
				escapes[l.prevCol()] = e
				isEscape = false
				b.WriteString(e.Value)
				continue loop
			} // "
		case 'b', 'e', 'f', 'n', 'r', 't':
			if isEscape {
				escape(EscCharacter, 0)
			}
		case 'x':
			if isEscape {
				parsedEscape(l.parseHexEsc(delim), 'x')
				continue loop
			}
		case 'u':
			if isEscape {
				parsedEscape(l.parseUnicodeEsc(delim), 'u')
				continue loop
			}
		case '\n':
			l.ResetPosition()
			if delim != '`' {
				// Invalid newline, just stop parsing
				unterm = true
				b.WriteRune(r)
				break loop
			}
			isNewline = true
			b.WriteRune(r)
			continue loop
		default:
			switch {
			case isEscape:
				escape(EscCharacter, ErrEscapeUnknown)
			case unicode.IsSpace(r):
				// Strip leading spaces from backtick string
				if isNewline && l.Pos.Col-1 <= pos.Col {
					continue loop
				}
			}
		}
		b.WriteRune(r)
		isNewline = false
	}
	str := string(delim) + b.String()

	return NewToken(pos, String, str).
		SetAttribute("escapes", escapes).
		SetAttribute("end", end).
		SetAttribute("quoteStyle", delim).
		SetAttribute("unterminated", unterm)
}
