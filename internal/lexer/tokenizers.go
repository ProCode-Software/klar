package lexer

import (
	"io"
	"unicode"
)

// Returns true if the error is EOF, otherwise panics
func handleReadError(err error) bool {
	if err != nil {
		if err == io.EOF {
			return true
		}
		panic(err)
	}
	return false
}

func (l *Lexer) ParseOperator() (TokenType, string) {
	op := l.TokenizeFunc(func(r rune, s *string) {
		switch r {
		// Only characters in a multichar operator
		case '#', '/', '*', '+', '-', '.', ':', '=', '!', '>', '<', '|', '&':
			*s += string(r)
		}
	})
	for len(op) >= 1 {
		if operator, is := OperatorMap[op]; is {
			if operator == Dot {
				l.Reader.UnreadRune()
				next, err := l.Reader.Peek(1)
				if handleReadError(err) {
					return Illegal, op
				}
				if unicode.IsDigit(rune(next[0])) {
					l.Reader.ReadRune()
					newToken := l.ParseNumber(l.Pos)
					return newToken.Kind, newToken.Source
				}
			}
			return operator, op
		}
		op = op[:len(op)-1] // Parsed too much characters: backup
	}
	return Illegal, op
}
func (l *Lexer) ParseLineComment() string {
	var shouldStop bool
	cmt := l.TokenizeFwdFunc(func(r rune, s *string) {
		if shouldStop {
			return
		}
		// Beginning // is already parsed
		if r == '\n' {
			l.ResetPosition()
			shouldStop = true
		}
		*s += string(r)
	})
	return "//" + cmt
}
func (l *Lexer) ParseBlockComment() string {
	cmtLevel := 1
	cmt := l.TokenizeFwdFunc(func(r rune, s *string) {
		if cmtLevel == 0 {
			return
		}
		if len(*s) > 1 {
			last := (*s)[len(*s)-1]
			if last == '*' && r == '/' {
				cmtLevel--
			} else if last == '/' && r == '*' {
				cmtLevel++
			}
		}
		*s += string(r)
	})
	return "/*" + cmt
}
func (l *Lexer) ParseNumber(pos Position) *Token {
	var (
		format     int
		isExponent bool
		isIllegal  bool
		isDot      bool
		errorType  int
	)
	digit := l.TokenizeFunc(func(r rune, lit *string) {
		s := unicode.ToLower(r)
		if *lit == "0" {
			switch s {
			case 'x':
				format = NumberFormatHexadecimal
				*lit += string(r)
				return
			case 'o':
				format = NumberFormatOctal
				*lit += string(r)
				return
			case 'b':
				format = NumberFormatBinary
				*lit += string(r)
				return
			default:
				format = NumberFormatDecimal
			}
		}
		switch s {
		case 'a', 'b', 'c', 'd', 'e', 'f':
			switch format {
			case NumberFormatHexadecimal:
				*lit += string(r)
			case NumberFormatDecimal:
				if s == 'e' && !isExponent {
					*lit += string(r)
				}
			default:
				// Hex letter or e on other format
				isIllegal = true
				errorType = ErrIntIncompatibleDigit
				*lit += string(r)
			}
		case '+', '-':
			if isExponent {
				*lit += string(r)
			}
		case '.':
			switch {
			// unread-peek-check-read
			case *lit == "": // .3
				format = NumberFormatDecimal
				*lit += string(r)
			case format != NumberFormatDecimal:
				errorType = ErrIntIncompatibleDigit
				isIllegal = true
				return
			default:
				l.Reader.UnreadRune()
				next, err := l.Reader.Peek(2)
				if handleReadError(err) {
					// Trailing decimal point at EOF
					l.Reader.ReadRune()
					*lit += string(r)
					return
				}
				if !unicode.IsDigit(rune(next[1])) {
					isDot = true
					l.Reader.ReadRune()
					return
				}
				// Normal decimal point
				l.Reader.ReadRune()
				*lit += string(r)
				return
			}
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if (*lit)[len(*lit)-1] == '_' {
				errorType = ErrIntMisplacedSeparator
				isIllegal = true
			}
			*lit += string(r)
		default:
			if !unicode.IsDigit(r) {
				return
			}
			isExponent = false
			if format == NumberFormatDecimal || format == NumberFormatHexadecimal ||
				(format == NumberFormatBinary && r <= '1') ||
				(format == NumberFormatOctal && r <= '7') {

				*lit += string(r)
				return
			}
			// Incompatible digit
			isIllegal = true
			errorType = ErrIntIncompatibleDigit
			*lit += string(r)
		}
	})
	switch {
	case digit == ".":
		return NewLexerToken(pos, Dot, digit)
	case isDot:
		// Not a number - may be 1...10
	case digit[len(digit)-1] == '_' || digit[0] == '_':
		isIllegal = true
		errorType = ErrIntIncompatibleDigit
		fallthrough
	case isIllegal:
		return NewLexerToken(pos, Illegal, digit)
	}
	return NewLexerToken(pos, Numeric, digit).
		SetAttribute("format", format).
		SetAttribute("invalid", isIllegal).
		SetAttribute("error", errorType)
}
func (l *Lexer) ParseIdentifier() (TokenType, string) {
	id := l.TokenizeFunc(func(r rune, lit *string) {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			*lit += string(r)
		}
	})
	if keyword, is := KeywordMap[id]; is {
		return keyword, id
	}
	return Identifier, id
}

func (l *Lexer) ParseString(pos Position) *Token {
	var (
		isEscape   bool
		shouldStop bool
		delim      rune
		err        int
		escapePos  = make([]Position, 0, 1)
		escapes    = make(map[Position]StringEscape)
	)
	insertEscape := func() {
		escapePos = append(escapePos, l.Pos)
	}
	str := l.TokenizeFunc(func(r rune, s *string) {
		if shouldStop {
			return
		}
		switch r {
		case '"', '\'', '`':
			if delim == 0 { // Unset
				delim = r
			} else if delim == r && !isEscape {
				shouldStop = true
			}
			*s += string(r)
			isEscape = false
		case '{':
			if delim == '"' {
				insertEscape()
			}
		case '\\':
			if delim != '`' {
				isEscape = !isEscape
				insertEscape()
			}
		case '\n':
			l.ResetPosition()
			if delim != '`' {
				// Invalid newline
				err = ErrStrUnterminated
				return
			}
			*s += string(r)
		default:
			isEscape = false
			*s += string(r)
		}
	})
	for _, p := range escapePos {
		escapes[p] = l.parseStringEscape(p, delim)
	}
	// Invalid if first character in string isn't the same as last (unterminated) (due to EOF)
	if str[0] != str[len(str)-1] {
		err = ErrStrUnterminated
	}
	return NewLexerToken(pos, String, str).
		SetAttribute("quoteStyle", delim).
		SetAttribute("error", err).
		SetAttribute("escapes", escapes)
}
