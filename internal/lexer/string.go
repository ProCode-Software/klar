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

type EscapeError int

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
	ErrorPosition int
}

func isHex(r rune) bool {
	return unicode.Is(unicode.ASCII_Hex_Digit, r)
}

func (l *Lexer) parseUnicodeEsc(delim rune) StringEscape {
	var (
		err    EscapeError
		errPos int
	)
	invalid := func(l int, reason EscapeError) {
		err, errPos = reason, l+2
	}
	esc := l.TokenizeFwdFunc(func(r rune, s *string) {
		l := len(*s)
		switch {
		case *s == "" && r == '{':
			*s += string(r)
		case l > 0 && (*s)[l-1] == '}':
			return
		case l > 6 && r != '}':
			invalid(l, ErrEscapeTooLong)
		case l < 2 && r == '}':
			invalid(l+1, ErrEscapeTooShort)
			*s += string(r)
		case r == '}', isHex(r):
			*s += string(r)
		case r == delim:
			invalid(l, ErrEscapeUnterm)
		default:
			invalid(l, ErrEscapeExpHex)
		}
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
		errPos  int
		invalid = func(l int) { err, errPos = ErrEscapeExpHex, l+2 }
	)
	esc := l.TokenizeFwdFunc(func(r rune, s *string) {
		l := len(*s)
		switch {
		case l == 2:
			return
		case isHex(r):
			*s += string(r)
		case r == delim:
			invalid(l)
			return
		default:
			invalid(l)
			*s += string(r)
		}
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
		errPos  int
		invalid = func(code EscapeError, l int) { err, errPos = code, l+2 }
	)
	esc := l.TokenizeFwdFunc(func(r rune, s *string) {
		l := len(*s)
		switch {
		case l == 0 && r == '}':
			invalid(ErrEscapeTooShort, l)
		case r == delim:
			invalid(ErrEscapeUnterm, l)
		case l > 0 && (*s)[l-1] == '}':
			return
		default:
			*s += string(r)
		}
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
		str, esc         string
		isEscape, unterm bool
		escapes          = make(map[int]StringEscape)
		escStart         int
	)
	escape := func(typ EscapeType, err EscapeError) {
		e := StringEscape{Type: typ, Value: `\` + esc, Invalid: err}
		if err > 0 {
			e.ErrorPosition = len(str) + 1
		}
		escapes[escStart], isEscape, esc = e, false, ""
	}
	parsedEscape := func(e StringEscape, p rune) {
		str += string(p) + e.Value[2:]
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
				str += string(r)
				break loop
			}
		case '\\':
			if isEscape {
				escape(EscCharacter, 0)
			} else if delim != '`' {
				isEscape, escStart = true, len(str)+1
			}
		case '{':
			// TODO
			str += string(r)
			continue
			if isEscape {
				if delim == '"' {
					escape(EscCharacter, 0)
				} else {
					escape(EscCharacter, ErrEscapeUnknown)
				}
			} else if delim == '"' {
				escapes[len(str)+1] = l.parseStrInterp(delim)
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
				str += string(r)
				break loop
			}
		default:
			if isEscape {
				escape(EscCharacter, ErrEscapeUnknown)
			}
		}
		str += string(r)
	}
	// TODO: check if closes due to EOF
	return NewToken(pos, String, string(delim)+str).
		SetAttribute("escapes", escapes).
		SetAttribute("quoteStyle", delim).
		SetAttribute("unterminated", unterm)
}
