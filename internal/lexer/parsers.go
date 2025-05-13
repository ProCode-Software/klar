package lexer

import (
	"fmt"
	"unicode"
)

func getOperatorType(op string) (TokenType, string) {
	for len(op) >= 1 {
		if operator, is := OperatorMap[op]; is {
			return operator, op
		}
		op = op[:len(op)-1] // Parsed too much characters: backup
	}
	return Illegal, op
}

func (l *Lexer) ParseOperator() (TokenType, string) {
	op := l.ParseFunc(func(r rune, s *string) {
		switch r {
		// Only characters in a multichar operator
		case '+', '-', '.', ':', '=', '!', '>', '<', '|', '&':
			*s += string(r)
		}
	})
	return getOperatorType(op)
}
func (l *Lexer) ParseLineComment() string {
	cmt := "/" + l.ParseFunc(func(r rune, s *string) {
		// ParseFunc backs up one rune, so / is reparsed
		if *s != "/" && r == '\n' {
			return
		}
		*s += string(r)
	})
	return cmt
}
func (l *Lexer) ParseBlockComment() string {
	cmt := "/" + l.ParseFunc(func(r rune, s *string) {
		// ParseFunc backs up one rune, so * is reparsed
		if *s != "*" && (*s)[len(*s)-1] == '*' && r == '/' {
			*s += string(r)
			return
		}
		*s += string(r)
	})
	return cmt
}
func (l *Lexer) ParseNumber(pos Position) *Token {
	var (
		format     int
		isExponent bool
		isIllegal  bool
		isDot      bool
		errorType  int
	)
	digit := l.ParseFunc(func(r rune, lit *string) {
		s := unicode.ToLower(r)
		if r == '.' {
			fmt.Printf("\033[33mNumber: %s\033[m\n", *lit)
		}
		if *lit == "0" {
			switch s {
			case 'x':
				format = Hexadecimal
				*lit += string(r)
				return
			case 'o':
				format = Octal
				*lit += string(r)
				return
			case 'b':
				format = Binary
				*lit += string(r)
				return
			default:
				format = Decimal
			}
		} else if *lit == "" && r == '.' { // .3
			format = Decimal
			*lit += string(r)
			return
		}
		if *lit == "." && !unicode.IsDigit(r) {
			// Parsed a dot
			l.Backup()
			return
		}
		switch s {
		case 'a', 'b', 'c', 'd', 'e', 'f':
			switch format {
			case Hexadecimal:
				*lit += string(r)
			case Decimal:
				if s == 'e' && !isExponent {
					*lit += string(r)
				}
			default:
				// Hex letter or e on other format
				isIllegal = true
				errorType = IntIncompatibleDigit
				*lit += string(r)
			}
		case '+', '-':
			if isExponent {
				*lit += string(r)
			}
		case '.':
			switch {
			case *lit == ".":
				format = Decimal
				*lit += string(r)
			case format == Decimal && (*lit)[len(*lit)-1] != '.':
				*lit += string(r)
			case (*lit)[len(*lit)-1] == '.':
				// An operator, not a number
				isDot = true
				return
			default:
				// Double decimal point or other
				isIllegal = true
				errorType = IntMultipleDot
				return
			}
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if (*lit)[len(*lit)-1] == '_' {
				errorType = IntMisplacedSeparator
				isIllegal = true
			}
			*lit += string(r)
		default:
			if !unicode.IsDigit(r) {
				return
			}
			isExponent = false
			if format == Decimal || format == Hexadecimal ||
				(format == Binary && r <= '1') ||
				(format == Octal && r <= '7') {

				*lit += string(r)
				return
			}
			// Incompatible digit
			isIllegal = true
			errorType = IntIncompatibleDigit
			*lit += string(r)
		}
	})
	switch {
	case digit == ".":
		return NewLexerToken(pos, Dot, digit)
	case isDot:
		
	case digit[len(digit)-1] == '_' || digit[0] == '_':
		isIllegal = true
		errorType = IntIncompatibleDigit
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
	id := l.ParseFunc(func(r rune, lit *string) {
		if r == '_' || unicode.IsLetter(r) || unicode.IsLetter(r) {
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
		isEscape bool
		delim    rune
		err      int
	)
	str := l.ParseFunc(func(r rune, s *string) {
		switch r {
		case '"', '\'', '`':
			if delim == 0 { // Unset
				delim = r
			} else if delim == r && !isEscape {
				return
			}
			*s += string(r)
		case '\\':
			isEscape = !isEscape
		case '\n':
			if delim != '`' {
				// Invalid newline
				err = StrUnterminated
				return
			}
			*s += string(r)
		default:
			*s += string(r)
		}
	}) + string(delim)
	// Invalid if first character in string isn't the same as last (unterminated)
	return NewLexerToken(pos, String, str).
		SetAttribute("quoteStyle", delim).
		SetAttribute("error", err)
}
