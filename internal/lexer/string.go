package lexer

import "unicode"

type StringEscapeType int

const (
	_ StringEscapeType = iota
	EscCharacter
	HexadecimalEscape
	EscUnicode
	EscInterpolation
)

type StringEscapeErrorType int

const (
	_ StringEscapeErrorType = iota // Not invalid
	ErrEscapeNotEnough
	ErrEscapeTooLong
	ErrEscapeExpectedChar
	ErrEscapeExpectedHexChar
	ErrEscapeUnknown
)

type StringEscape struct {
	Type          StringEscapeType
	Value         string
	Invalid       StringEscapeErrorType
	ErrorPosition int
}

func isHex(r rune) bool {
	return unicode.Is(unicode.ASCII_Hex_Digit, r)
}

func (l *Lexer) parseUnicodeEsc(delim rune) StringEscape {
	var (
		err    StringEscapeErrorType
		errPos int
	)
	invalid := func(l int, reason StringEscapeErrorType) {
		err, errPos = ErrEscapeExpectedHexChar, l+2
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
			invalid(l, ErrEscapeNotEnough)
			*s += string(r)
		case r == '{', isHex(r):
			*s += string(r)
		case r == delim:
			invalid(l, ErrEscapeExpectedChar)
		default:
			invalid(l, ErrEscapeExpectedHexChar)
		}
	})
	// TODO: check length and closing due to EOF
	return StringEscape{
		Type:          EscUnicode,
		Value:         "\\u" + esc,
		Invalid:       err,
		ErrorPosition: errPos,
	}
}
func (l *Lexer) parseHexEsc(delim rune) StringEscape {
	var (
		err    StringEscapeErrorType
		errPos int
	)
	invalid := func(l int) { err, errPos = ErrEscapeExpectedHexChar, l+2 }
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
		Type:          EscUnicode,
		Value:         "\\x" + esc,
		Invalid:       err,
		ErrorPosition: errPos,
	}
}
func (l *Lexer) parseStrInterp(delim rune) StringEscape {
	var (
		err    StringEscapeErrorType
		errPos int
	)
	invalid := func(code StringEscapeErrorType, l int) {
		err, errPos = code, l+2
	}
	esc := l.TokenizeFwdFunc(func(r rune, s *string) {
		l := len(*s)
		switch {
		case l == 0 && r == '}':
			invalid(ErrEscapeNotEnough, l)
		case r == delim:
			invalid(ErrEscapeExpectedChar, l)
		case (*s)[l-1] == '}':
			return
		default:
			*s += string(r)
		}
	})
	// TODO: check length and closing due to EOF
	return StringEscape{
		Type:          EscUnicode,
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
	escape := func(typ StringEscapeType, err StringEscapeErrorType) {
		e := StringEscape{Type: typ, Value: esc, Invalid: err}
		if err > 0 {
			e.ErrorPosition = len(str) + 1
		}
		escapes[escStart], isEscape, esc = e, false, ""
	}
	for {
		r, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			unterm = true
			break
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
				break
			}
		case '\\':
			if isEscape {
				escape(EscCharacter, 0)
			} else if delim != '`' {
				isEscape, escStart = true, len(str)+1
			}
		case '{':
			if delim == '"' && !isEscape {
				escapes[len(str)+1] = l.parseStrInterp(delim)
			} else if isEscape {
				escape(EscCharacter, ErrEscapeUnknown)
			}
		case 'b', 'e', 'f', 'n', 'r', 't':
			if isEscape && esc == `\` {
				isEscape = false
				escape(EscCharacter, 0)
			}
		case 'x':
			escapes[escStart] = l.parseHexEsc(delim)
			isEscape = false
		case 'u':
			escapes[escStart] = l.parseUnicodeEsc(delim)
			isEscape = false
		case '\n':
			l.ResetPosition()
			if delim != '`' {
				// Invalid newline, just stop parsing
				unterm = true
				str += string(r)
				break
			}
		default:
			if isEscape {
				escape(EscCharacter, ErrEscapeUnknown)
			}
		}
		str += string(r)
	}
	// TODO: check if closes due to EOF
	return NewToken(pos, String, str).
		SetAttribute("escapes", escapes).
		SetAttribute("quoteStyle", delim).
		SetAttribute("unterminated", unterm)
}
