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

func (l *Lexer) ParseOperator(r rune) (TokenType, string) {
	s := string(r)
	if _, ok := OperatorMap[s]; !ok {
		return Illegal, s
	}
	next, err := l.Reader.Peek(1)
	if handleReadError(err) {
		return OperatorMap[s], s
	}
	full := s + string(next)
	if op, ok := OperatorMap[full]; ok {
		l.Reader.ReadRune()
		l.Pos.Col++
		return op, full
	}
	return OperatorMap[s], s
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

func (l *Lexer) ParseBlockComment() (string, Position) {
	cmtLevel := 1
	var endPos Position
	cmt := l.TokenizeFwdFunc(func(r rune, s *string) {
		switch {
		case cmtLevel == 0:
			endPos = l.Pos
			return
		case r == '\n':
			l.ResetPosition()
		case len(*s) > 1:
			last := (*s)[len(*s)-1]
			if last == '*' && r == '/' {
				cmtLevel--
			} else if last == '/' && r == '*' {
				cmtLevel++
			}
		}
		*s += string(r)
	})
	return "/*" + cmt, endPos
}

const (
	_ = iota
	ErrIntMisplacedSeparator
	ErrIntIncompatibleDigit
	ErrIntIllegalExponent
	ErrStrUnterminated
)

func (l *Lexer) ParseNumber(pos Position) *Token {
	var (
		format, errorType, errPos   int
		isExp, isIllegal, isDecimal bool
	)
	newError := func(code int, lit *string) {
		errorType = code
		errPos = len(*lit)
		isIllegal = true
	}
	digit := l.TokenizeFunc(func(r rune, lit *string) {
		lower := unicode.ToLower(r)
		if *lit == "0" {
			switch lower {
			case 'x':
				format = NumberFormatHex
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
		switch lower {
		case 'e':
			if format == NumberFormatDecimal && !isExp {
				*lit += string(r)
				isExp = true
			}
			fallthrough
		case 'a', 'b', 'c', 'd', 'f':
			switch format {
			case NumberFormatHex:
				*lit += string(r)
			default:
				// Hex letter or e on other format
				newError(ErrIntIncompatibleDigit, lit)
				// *lit += string(r)
			}
		case '+', '-':
			if isExp {
				*lit += string(r)
			}
		case '.':
			switch {
			case isDecimal:
				return
			case *lit == "": // .3
				format = NumberFormatDecimal
				*lit += string(r)
				isDecimal = true
			case format != NumberFormatDecimal:
				newError(ErrIntIncompatibleDigit, lit)
				fallthrough
			default:
				l.Reader.UnreadRune()
				next, err := l.Reader.Peek(2)
				l.Reader.ReadRune()
				if handleReadError(err) || unicode.IsDigit(rune(next[1])) {
					// Trailing decimal point at EOF
					isDecimal = true
					*lit += string(r)
					return
				}
			}
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if (*lit)[len(*lit)-1] == '_' {
				newError(ErrIntMisplacedSeparator, lit)
			}
			*lit += string(r)
		default:
			switch {
			case !unicode.IsDigit(r):
				return
			case format == NumberFormatDecimal, format == NumberFormatHex,
				(format == NumberFormatBinary && r <= '1'),
				(format == NumberFormatOctal && r <= '7'):
				*lit += string(r)
			default:
				newError(ErrIntIncompatibleDigit, lit)
				*lit += string(r)
			}
		}
	})
	// Last digit is separator
	if digit[len(digit)-1] == '_' {
		newError(ErrIntMisplacedSeparator, &digit)
		errPos = len(digit) - 1
	}
	return NewToken(pos, Numeric, digit).
		SetAttribute("format", format).
		SetAttribute("invalid", isIllegal).
		SetAttribute("errorPos", errPos).
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
