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
	esc := l.TokenizeFunc(func(r rune, b *Builder) bool {
		l := b.Len()
		switch {
		case l == 0 && r == '{':
			b.WriteRune(r)
		case l > 0 && last == '}':
			return false
		case l > 6 && r != '}':
			invalid(ErrEscapeTooLong)
		case l < 2 && r == '}':
			invalid(ErrEscapeTooShort)
			b.WriteRune(r)
		case r == '}', isHex(r):
			b.WriteRune(r)
		case r == delim:
			invalid(ErrEscapeUnterm)
		default:
			invalid(ErrEscapeExpHex)
		}
		last = r
		return true
	})
	// TODO: check length and closing due to EOF
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
		invalid = func() { err, errPos = ErrEscapeExpHex, l.Pos }
	)
	esc := l.TokenizeFunc(func(r rune, b *Builder) bool {
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
	})
	// TODO: check length and closing due to EOF
	return StringEscape{
		Type:          EscHex,
		Value:         `\x` + esc,
		Invalid:       err,
		ErrorPosition: errPos,
	}
}

func (l *Lexer) parseStrInterp(delim rune) StringEscape {
	var (
		err     EscapeError
		errPos  Position
		invalid = func(code EscapeError) { err, errPos = code, l.Pos }
		last    rune
	)
	esc := l.TokenizeFunc(func(r rune, b *Builder) bool {
		l := b.Len()
		switch {
		case l == 0 && r == '}':
			invalid(ErrEscapeTooShort)
		case r == delim:
			invalid(ErrEscapeUnterm)
		case l > 0 && last == '}':
			return false
		default:
			b.WriteRune(r)
		}
		last = r
		return true
	})
	// TODO: check length and closing due to EOF
	return StringEscape{
		Type:          EscInterpolation,
		Value:         "{" + esc,
		Invalid:       err,
		ErrorPosition: errPos,
	}
}

// Beginning is already parsed
func (l *Lexer) ParseString(pos Position, delim rune) *Token {
	var (
		esc              string
		b                Builder
		isEscape, unterm bool
		escapes          = make(EscapeMap)
		escStart         Position
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
				break loop
			}
		case '\\':
			if isEscape {
				escape(EscCharacter, 0)
			} else if delim != '\'' {
				isEscape, escStart = true, l.Pos
			}
		case '{':
			// TODO
			b.WriteRune(r)
			if isEscape {
				escape(EscCharacter, 0)
			} else if delim != '\'' {
				escapes[l.Pos] = l.parseStrInterp(delim)
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
		default:
			if isEscape {
				escape(EscCharacter, ErrEscapeUnknown)
			}
		}
		b.WriteRune(r)
	}
	// TODO: check if closes due to EOF
	return NewToken(pos, String, string(delim)+b.String()).
		SetAttribute("escapes", escapes).
		SetAttribute("quoteStyle", delim).
		SetAttribute("unterminated", unterm)
}
