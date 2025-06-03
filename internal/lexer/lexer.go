package lexer

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Builder = strings.Builder

type Position struct {
	Line, Col int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
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
		l.Pos.Col++
		if handleReadError(err) {
			return NewToken(pos, EOF, "")
		}

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
				tok := l.ParseBlockComment(pos)
				if !l.IncludeComments {
					continue
				}
				return tok
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
					return NewToken(pos, Ellipsis, "...")
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

// BackupTokenizeFunc backs up the lexer before calling TokenizeFunc.
func (l *Lexer) BackupTokenizeFunc(fn func(rune, *Builder) bool) string {
	l.Backup()
	return l.TokenizeFunc(fn)
}

func (l *Lexer) TokenizeFunc(fn func(rune, *Builder) bool) string {
	var b Builder
	for {
		r, _, err := l.Reader.ReadRune()
		l.Pos.Col++
		if handleReadError(err) {
			return b.String()
		}
		if r == '\n' {
			l.ResetPosition()
		}
		if !fn(r, &b) {
			l.Backup()
			return b.String()
		}
	}
}

// TokenizeFunc with a callback if the lexer reaches EOF.
func (l *Lexer) TokenizeEOFFunc(
	fn func(rune, *Builder) bool,
	onEOF func(),
) string {
	var b Builder
	for {
		rune, _, err := l.Reader.ReadRune()
		l.Pos.Col++
		if handleReadError(err) {
			onEOF()
			return b.String()
		}
		if rune == '\n' {
			l.ResetPosition()
		}
		if !fn(rune, &b) {
			l.Backup()
			return b.String()
		}
	}
}
