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
		case '.', '!', '+', ':', '-', '&', '|', '=', '>', '<', '/', '#':
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
		case ' ':
			continue
		// Single-character tokens and operators
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
		case r == 0xfeff:
			if pos.Line == 1 && pos.Col == 1 {
				continue
			}
			fallthrough
		default:
			attrs := attrs{"length": uint32(1)}
			if r == utf8.RuneError {
				attrs["invalidCharacter"] = struct{}{}
			}
			return NewToken(pos, Illegal, string(r)).withAttrs(attrs)
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
	for r := range l.NewTokenizer(true).Tokenize {
		if r != char {
			break
		}
		n++
	}
	return
}

func (l *Lexer) Reset() {
	l.Reader.Reset(nil)
	l.Pos.Line = 1
	l.Pos.Col = 1
	l.Flags = 0
}

// Utils
// ================

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

// Tokenizer
// ================
type Tokenizer struct {
	Builder    strings.Builder
	BackupLast bool
	eof        bool
	*Lexer
}

func (l *Lexer) NewTokenizer(backupLast bool) *Tokenizer {
	return &Tokenizer{Builder: strings.Builder{}, BackupLast: backupLast, Lexer: l}
}

func (t *Tokenizer) Tokenize(yield func(rune, *Builder) bool) {
	for {
		r, _, err := t.Reader.ReadRune()
		t.Pos.Col++
		if handleReadError(err) {
			t.eof = true
			return
		}
		if !yield(r, &t.Builder) {
			if t.BackupLast {
				t.Backup()
			}
			return
		}
		if r == '\n' {
			t.ResetPosition()
		}
	}
}

// EOF reports whether the lexer has reached the end of the input.
func (t *Tokenizer) EOF() bool { return t.eof }

// String returns the string representation of the accumulated tokens.
func (t *Tokenizer) String() string { return t.Builder.String() }

func (t *Tokenizer) Reset(backupLast bool) {
	t.Builder.Reset()
	t.BackupLast = backupLast
	t.eof = false
}
