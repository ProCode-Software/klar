package lexer

import (
	"bufio"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	Pos             Position
	Reader          *bufio.Reader
	ExcludeComments bool
}

func NewLexer(reader io.Reader) *Lexer {
	return &Lexer{Position{1, 1}, bufio.NewReader(reader), false}
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
			// Strings
			return l.ReadString(pos, r)
		case '.', '!', '+', ':', '-', '&', '|', '=', '>', '<', '/', '#', '*', '%', '^':
			// Multi-character operators
			var (
				typ, val = l.ReadOperator(r)
				tok      *Token
			)
			switch typ {
			default:
				return NewToken(pos, typ, val)
			case LineComment:
				tok = l.ReadLineComment(pos)
			case BlockComment:
				tok = l.ReadBlockComment(pos)
			case Hashbang:
				tok = l.ReadShebang(pos)
			case Regex:
				tok = l.ReadRegex(pos)
			}
			if l.ExcludeComments {
				continue
			}
			return tok
		case ' ':
			continue
		// Single-character tokens and operators
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
		case '@':
			return NewToken(pos, At, "@")
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case IsDigit(r):
			return l.ReadNumber(pos, r)
		case unicode.IsLetter(r), r == '_':
			return l.ReadIdentifier(pos, r)
		case r == 0xfeff:
			// Byte order mark
			if pos.Line == 1 && pos.Col == 1 {
				continue
			}
			fallthrough
		default:
			// Invalid UTF-8
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

func (l *Lexer) Reset() {
	l.Reader.Reset(nil)
	l.Pos.Line = 1
	l.Pos.Col = 1
	l.ExcludeComments = false
}

// Utils
// ================

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

func IsDigit(r rune) bool { return r >= '0' && r <= '9' }

// IsIdent reports whether r is the beginning of an identifier
func IsIdent(r rune) bool { return r == '_' || unicode.IsLetter(r) }

func IsASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func IsHex(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
}

// Tokenizer
// ================
type Tokenizer struct {
	Builder    strings.Builder
	BackupLast bool
	eof        bool
	endPos     Position
	*Lexer
}

func (l *Lexer) NewTokenizer(backupLast bool) *Tokenizer {
	return &Tokenizer{Builder: strings.Builder{}, BackupLast: backupLast, Lexer: l}
}

func (t *Tokenizer) Tokenize(yield func(rune, *strings.Builder) bool) {
	for {
		r, _, err := t.Reader.ReadRune()
		t.Pos.Col++
		switch {
		case handleReadError(err):
			t.eof = true
			t.endPos = t.Pos
			return
		case !yield(r, &t.Builder):
			if t.BackupLast {
				t.Backup()
			}
			t.endPos = t.Pos
			return
		case r == '\n':
			t.ResetPosition()
		}
	}
}

// EOF reports whether the lexer has reached the end of the input.
func (t *Tokenizer) EOF() bool { return t.eof }

// String returns the string representation of the accumulated tokens.
func (t *Tokenizer) String() string { return t.Builder.String() }

// EndPos returns the position of the last read character.
func (t *Tokenizer) EndPos() Position { return t.endPos }

func (t *Tokenizer) Reset(backupLast bool) {
	t.Builder.Reset()
	t.BackupLast = backupLast
	t.eof = false
}

// ResetKeepBuilder resets the tokenizer without clearing the builder or end position.
func (t *Tokenizer) ResetKeepBuilder(backupLast bool) {
	t.BackupLast = backupLast
	t.eof = false
}

// RuneReader is an interface for reading runes from a stream.
type RuneReader interface {
	AdvanceRune() (rune, error)
	CurrRune() (rune, error)
	PeekRune() (rune, error)
	Position() Position
}

func (l *Lexer) AdvanceRune() (rune, error) {
	r, _, err := l.Reader.ReadRune()
	if err != nil {
		return 0, err
	}
	l.Pos.Col++
	if r == '\n' {
		l.ResetPosition()
	}
	return r, nil
}

func (l *Lexer) CurrRune() (rune, error) {
	b, err := l.Reader.Peek(4)
	if err != nil && len(b) == 0 {
		return 0, err
	}
	r, _ := utf8.DecodeRune(b)
	return r, nil
}

func (l *Lexer) PeekRune() (rune, error) {
	b, err := l.Reader.Peek(8)
	if err != nil && len(b) == 0 {
		return 0, err
	}
	_, size1 := utf8.DecodeRune(b)
	if len(b) <= size1 {
		return 0, io.EOF
	}
	r2, _ := utf8.DecodeRune(b[size1:])
	return r2, nil
}

func (l *Lexer) Position() Position {
	return l.Pos
}
