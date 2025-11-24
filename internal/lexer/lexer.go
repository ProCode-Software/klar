package lexer

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/char"
)

type Builder = strings.Builder

type Position struct {
	Line, Col uint32
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

type Flags uint8

const (
	IncludeComments Flags = 1 << iota // For documentation parsers
)

type Lexer struct {
	Pos    Position
	Reader *bufio.Reader
	Flags  Flags
}

func NewLexer(reader io.Reader, flags Flags) *Lexer {
	return &Lexer{Position{1, 1}, bufio.NewReader(reader), flags}
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
			return l.ReadString(pos, r, 0)
		case '.':
			next, isEOF := l.BackupPeek()
			if isEOF {
				return NewToken(pos, Dot, ".")
			}
			if IsDigit(rune(next)) {
				return l.ReadNumber(pos)
			}
			fallthrough
		case '!', '+', ':', '-', '&', '|', '=', '>', '<', '/', '#':
			// Multi-character operators
			var (
				typ, val = l.ReadOperator(r)
				tok      *Token
			)
			switch typ {
			default:
				return NewToken(pos, typ, val)
			// Just change position
			case LineComment:
				tok = l.ReadLineComment(pos)
			case BlockComment:
				tok = l.ReadBlockComment(pos)
			case Hashbang:
				tok = l.ReadShebang(pos)
			}
			if l.Flags&IncludeComments == 0 {
				continue
			}
			return tok
		// Single-character tokens and operators
		case '\\':
			return NewToken(pos, Backslash, `\`)
		case '@':
			next, isEOF := l.BackupPeek()
			if !isEOF {
				switch next {
				case '"', '`', '\'':
					next := rune(next)
					return l.ReadString(pos, next, l.ReadAll(next))
				case '/':
					next := rune(next)
					return l.ReadRegex(pos, l.ReadAll(next))
				}
			}
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
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case IsDigit(r):
			return l.ReadNumber(pos)
		case unicode.IsLetter(r), r == '_':
			return l.ReadIdentifier(pos)
		default:
			return NewToken(pos, Illegal, string(r)).withAttrs(attrs{
				"length": uint32(utf8.RuneLen(r)),
			})
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
	return l.TokenizeEOFFunc(fn, nil)
}

// TokenizeFunc with a callback if the lexer reaches EOF.
func (l *Lexer) TokenizeEOFFunc(
	fn func(rune, *Builder) bool,
	onEOF func(),
) string {
	var b Builder
	for {
		r, _, err := l.Reader.ReadRune()
		l.Pos.Col++
		if handleReadError(err) {
			if onEOF != nil {
				onEOF()
			}
			return b.String()
		}
		if !fn(r, &b) {
			l.Backup()
			return b.String()
		}
		if r == '\n' {
			l.ResetPosition()
		}
	}
}

func (l *Lexer) prevCol() Position {
	return Position{Line: l.Pos.Line, Col: l.Pos.Col - 1}
}

// BackupPeek backs up the lexer, peeks n + 1 bytes, re-reads the rune, and returns n bytes.
// The only error returned is EOF
func (l *Lexer) BackupPeek() (b byte, eof bool) {
	if err := l.Reader.UnreadRune(); err != nil {
		panic(err)
	}
	next, err := l.Reader.Peek(2)
	l.Reader.ReadRune()
	if handleReadError(err) {
		return 0, true
	}
	return next[1], false
}

func (l *Lexer) PeekN(n int) (b []byte, eof bool) {
	next, err := l.Reader.Peek(n)
	if handleReadError(err) {
		return next, true
	}
	return next, false
}

func (l *Lexer) ReadAll(char rune) (n int) {
	l.TokenizeFunc(func(r rune, b *Builder) bool {
		if r != char {
			return false
		}
		n++
		return true
	})
	return
}

func (l *Lexer) Free() {
	l.Reader.Reset(nil)
	l.Pos.Line = 1
	l.Pos.Col = 1
	l.Flags = 0
}

func IsDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// IsIdent reports whether r is the beginning of an identifier
func IsIdent(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// r should be an ASCII character
func repeat(prefix, r rune, n int) string {
	return string(prefix) + string(char.Repeat(byte(r), n))
}
