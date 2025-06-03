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
	cmt := l.TokenizeFunc(func(r rune, b *Builder) bool {
		if shouldStop {
			return false
		}
		// Beginning // is already parsed
		if r == '\n' {
			shouldStop = true
		}
		b.WriteRune(r)
		return true
	})
	return "//" + cmt
}

func (l *Lexer) ParseBlockComment(pos Position) *Token {
	var (
		endPos   Position
		cmtLevel = 1
		unterm   bool
		b        Builder
		last     rune
	)
	b.WriteString("/*")
loop:
	for {
		r, _, err := l.Reader.ReadRune()
		l.Pos.Col++
		if handleReadError(err) {
			unterm = true
			endPos = l.Pos
			break loop
		}
		switch {
		case r == '/' && last == '*':
			if b.Len() > 2 {
				cmtLevel--
			}
			if cmtLevel == 0 {
				b.WriteRune(r)
				endPos = l.nextCol()
				break loop
			}
		case r == '*' && last == '/':
			cmtLevel++
		case r == '\n':
			l.ResetPosition()
		}
		last = r
		b.WriteRune(r)
	}

	return NewToken(pos, BlockComment, b.String()).
		SetAttribute("unterm", unterm).SetAttribute("end", endPos)
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
		last                        rune
	)
	newError := func(code int, b *Builder) {
		errorType = code
		if b != nil {
			errPos = b.Len()
		}
		isIllegal = true
	}
	digit := l.BackupTokenizeFunc(func(r rune, b *Builder) bool {
		lower := unicode.ToLower(r)
		if b.String() == "0" {
			switch lower {
			case 'x':
				format = NumberFormatHex
				b.WriteRune(r)
			case 'o':
				format = NumberFormatOctal
				b.WriteRune(r)
			case 'b':
				format = NumberFormatBinary
				b.WriteRune(r)
			default:
				format = NumberFormatDecimal
				return false
			}
			return true
		}
		switch lower {
		case 'e':
			if format == NumberFormatDecimal && !isExp {
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b)
					errPos--
				}
				b.WriteRune(r)
				isExp = true
				return true
			}
			fallthrough
		case 'a', 'b', 'c', 'd', 'f':
			switch format {
			case NumberFormatHex:
				b.WriteRune(r)
			default:
				// Hex letter or e on other format
				newError(ErrIntIncompatibleDigit, b)
				// b.WriteRune(r)
			}
		case '+', '-':
			if !isExp {
				return false
			}
			b.WriteRune(r)
		case '.':
			switch {
			case isDecimal:
				return false
			case b.Len() == 0: // .3
				format = NumberFormatDecimal
				b.WriteRune(r)
				isDecimal = true
			case format != NumberFormatDecimal:
				newError(ErrIntIncompatibleDigit, b)
				fallthrough
			default:
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b)
					errPos--
				}
				l.Reader.UnreadRune()
				next, err := l.Reader.Peek(2)
				l.Reader.ReadRune()
				if handleReadError(err) || next[1] == '\n' ||
					unicode.IsDigit(rune(next[1])) {
					// Trailing decimal point at EOF
					isDecimal = true
					b.WriteRune(r)
					return true
				}
			}
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if last == '_' {
				newError(ErrIntMisplacedSeparator, b)
			} else if format == NumberFormatDecimal && !unicode.IsDigit(rune(last)) {
				newError(ErrIntMisplacedSeparator, b)
			}
			b.WriteRune(r)
		default:
			switch {
			case !unicode.IsDigit(r):
				return false
			case
				format == NumberFormatDecimal,
				format == NumberFormatHex,
				format == NumberFormatBinary && r <= '1',
				format == NumberFormatOctal && r <= '7':
				b.WriteRune(r)
			default:
				newError(ErrIntIncompatibleDigit, b)
				b.WriteRune(r)
			}
		}
		last = r
		return true
	})
	// Last digit is separator
	if digit[len(digit)-1] == '_' {
		newError(ErrIntMisplacedSeparator, nil)
		errPos = len(digit) - 1
	}
	return NewToken(pos, Numeric, digit).
		SetAttribute("format", format).
		SetAttribute("invalid", isIllegal).
		SetAttribute("errorPos", errPos).
		SetAttribute("error", errorType)
}

func (l *Lexer) ParseIdentifier() (TokenType, string) {
	id := l.BackupTokenizeFunc(func(r rune, b *Builder) bool {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			return true
		}
		return false
	})
	if keyword, is := KeywordMap[id]; is {
		return keyword, id
	}
	return Identifier, id
}
