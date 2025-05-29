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
	Pos             Position
	Reader          *bufio.Reader
	IncludeComments bool // Only for doc parsers
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{
		Position{1, 1}, bufio.NewReader(reader), false,
	}
}

func (l *Lexer) Tokenize() *Token {
	for {
		pos := l.Pos
		r, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			return NewToken(pos, EOF, "")
		}
		l.Pos.Col++

		switch r {
		case '\n':
			l.ResetPosition()
			return NewToken(pos, Newline, "\n")
		case '"', '\'', '`':
			return l.ParseString(pos, r)
		case '!', '+', ':', '-', '&', '|', '=', '>', '<', '/', '#':
			// Multi-character operators
			tt, val := l.ParseOperator(r)
			switch tt {
			default:
				return NewToken(pos, tt, val)
			// Just change position
			case LineComment:
				src := l.ParseLineComment()
				if !l.IncludeComments {
					continue
				}
				return NewToken(pos, LineComment, src)
			case BlockComment:
				src, endPos := l.ParseBlockComment()
				if !l.IncludeComments {
					continue
				}
				return NewToken(pos, BlockComment, src).SetAttribute("end", endPos)
			}
		// Single-character operators
		case '@':
			return NewToken(pos, At, "@")
		case '*':
			return NewToken(pos, Asterisk, "*")
		case '%':
			return NewToken(pos, Percent, "%")
		case '^':
			return NewToken(pos, Caret, "^")
		case '(':
			return NewToken(pos, LeftParenthesis, "(")
		case ')':
			return NewToken(pos, RightParenthesis, ")")
		case '{':
			return NewToken(pos, LeftCurlyBrace, "{")
		case '}':
			return NewToken(pos, RightCurlyBrace, "}")
		case '[':
			return NewToken(pos, LeftBracket, "[")
		case ']':
			return NewToken(pos, RightBracket, "]")
		case ',':
			return NewToken(pos, Comma, ",")
		case '?':
			return NewToken(pos, Question, "?")
		case '.':
			if err := l.Reader.UnreadRune(); err != nil {
				panic(err)
			}
			next, err := l.Reader.Peek(2)
			l.Reader.ReadRune()
			if handleReadError(err) {
				return NewToken(pos, Dot, ".")
			}
			if unicode.IsDigit(rune(next[1])) {
				return l.ParseNumber(pos)
			}
			if next[1] == '.' {
				l.Reader.ReadRune()
				l.Pos.Col++
				next, err = l.Reader.Peek(1)
				if handleReadError(err) || next[0] != '.' {
					return NewToken(pos, Illegal, "..")
				} else {
					l.Reader.ReadRune()
					l.Pos.Col++
					return NewToken(pos, Spread, "...")
				}
			}
			return NewToken(pos, Dot, ".")
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case unicode.IsDigit(r):
			return l.ParseNumber(pos)
		case unicode.IsLetter(r), r == '_':
			tt, val := l.ParseIdentifier()
			return NewToken(pos, tt, val)
		default:
			return NewToken(pos, Illegal, string(r))
		}
	}
}

func (l *Lexer) ResetPosition() {
	l.Pos.Col = 1
	l.Pos.Line++
}

func (l *Lexer) Backup() {
	if err := l.Reader.UnreadRune(); err != nil {
		panic(err)
	}
	l.Pos.Col--
}

func (l *Lexer) TokenizeFunc(fn func(rune, *string)) (literal string) {
	l.Backup()
	literal = l.TokenizeFwdFunc(fn)
	return
}

// TokenizeFunc but doesn't backup first
func (l *Lexer) TokenizeFwdFunc(fn func(rune, *string)) (literal string) {
	var oldLit string
	for {
		rune, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			return
		}
		l.Pos.Col++
		oldLit = literal
		fn(rune, &literal)
		if literal == oldLit {
			l.Backup()
			return
		}
	}
}
