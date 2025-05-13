package lexer

import (
	"bufio"
	"io"
	"unicode"
)

type Position struct {
	Line, Col int
}

type Lexer struct {
	Pos    Position
	Reader *bufio.Reader
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{Position{1, 0}, bufio.NewReader(reader)}
}

func (l *Lexer) Parse() (Position, TokenType, string) {
	for {
		pos := l.Pos
		rune, _, err := l.Reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return pos, EOF, ""
			}
			panic(err)
		}
		l.Pos.Col++

		switch rune {
		case '\n':
			l.ResetPosition()
			return pos, Newline, "\n"
		case '"', '\'', '`':
			return pos, String, l.ParseString()
		case '+', ':', '-', '.', '&', '|', '=', '>', '<', '/':
			// Multi-character operators
			tt, val := l.ParseOperator()
			// Keep going if it's a dot
			if !(tt == Illegal && val == ".") {
				return pos, tt, val
			}
		// Single-character operators
		case '*':
			return pos, Times, "*"
		case '%':
			return pos, Modulo, "%"
		case '^':
			return pos, Exponent, "^"
		case '!':
			return pos, LogicalNot, "!"
		case '(':
			return pos, LeftParenthesis, "("
		case ')':
			return pos, RightParenthesis, ")"
		case '{':
			return pos, LeftCurlyBrace, "{"
		case '}':
			return pos, RightCurlyBrace, "}"
		case '[':
			return pos, LeftBracket, "["
		case ']':
			return pos, RightBracket, "]"
		}
		switch {
		case unicode.IsSpace(rune):
			continue
		case unicode.IsDigit(rune) || rune == '.':
			// Also covers dot
			tt, val := l.ParseNumber()
			return pos, tt, val
		case unicode.IsLetter(rune), rune == '_':
			tt, val := l.ParseIdentifier()
			return pos, tt, val
		}
	}
}

func (l *Lexer) ResetPosition() {
	l.Pos.Col = 0
	l.Pos.Line++
}

func (l *Lexer) Backup() {
	if err := l.Reader.UnreadRune(); err != nil {
		panic(err)
	}
	l.Pos.Col--
}

func (l *Lexer) ParseFunc(fn func(rune, *string)) (literal string) {
	l.Backup()
	var oldLit string
	for {
		rune, _, err := l.Reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return literal
			}
			panic(err)
		}
		l.Pos.Col++
		oldLit = literal
		fn(rune, &literal)
		if literal == oldLit {
			return
		}
	}
}

func (l *Lexer) ParseOperator() (TokenType, string) {
	op := l.ParseFunc(func(r rune, s *string) {
		switch r {
		// Only characters in a multichar operator
		case '+', '-', '.', ':', '=', '!', '>', '<', '|', '&':
			*s += string(r)
		}
	})
	if operator, is := OperatorMap[op]; is {
		return operator, op
	}
	return Illegal, op // Not an operator or missing space
}
func (l *Lexer) ParseNumber() (TokenType, string) {
	var format int
	var isExponent, isIllegal bool
	digit := l.ParseFunc(func(r rune, lit *string) {
		s := unicode.ToLower(r)
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
		}
		switch s {
		case 'a', 'b', 'c', 'd', 'e', 'f':
			if format == Hexadecimal {
				*lit += string(r)
			} else if format == Decimal && s == 'e' && !isExponent {
				*lit += string(r)
				isExponent = true
			} else {
				// Hex digit or e on octal or binary format
				isIllegal = true
				*lit += string(r)
			}
		case '+', '-':
			if isExponent {
				*lit += string(r)
			}
		case '.':
			if format == Decimal {
				*lit += string(r)
			}
		default:
			if unicode.IsDigit(r) {
				isExponent = false
				if format == Decimal || format == Hexadecimal ||
					(format == Binary && r <= '1') ||
					(format == Octal && r <= '7') {

					*lit += string(r)
					return
				} else {
					// Incompatible digit
					isIllegal = true
					*lit += string(r)
				}
			}
		}
	})
	if digit == "." {
		return Dot, "."
	}
	if isIllegal {
		return Illegal, digit
	}
	return Numeric, digit
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
func (l *Lexer) ParseString() string {
	var isEscape bool
	var delim rune
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
				return
			}
		default:
			*s += string(r)
		}
	})
	// Invalid if first character in string isn't the same as last (unterminated)
	return str
}
