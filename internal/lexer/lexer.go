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
		rune, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			return NewLexerToken(pos, EOF, "")
		}
		l.Pos.Col++

		switch rune {
		case '\n':
			l.ResetPosition()
			return NewLexerToken(pos, Newline, "\n")
		case '"', '\'', '`':
			return l.ParseString(pos)
		case '+', ':', '-', '.', '&', '|', '=', '>', '<', '/':
			// Multi-character operators
			tt, val := l.ParseOperator()
			// Skip comments, just change position
			if tt == LineComment {
				src := l.ParseLineComment()
				if !l.IncludeComments {
					continue
				}
				return NewLexerToken(pos, LineComment, src)
			}
			if tt == BlockComment {
				src := l.ParseBlockComment()
				if !l.IncludeComments {
					continue
				}
				return NewLexerToken(pos, BlockComment, src)
			}
			// Keep going if it's a dot
			if !(tt == Illegal && val == ".") {
				return NewLexerToken(pos, tt, val)
			}
		// Single-character operators
		case '*':
			return NewLexerToken(pos, Times, "*")
		case '%':
			return NewLexerToken(pos, Modulo, "%")
		case '^':
			return NewLexerToken(pos, Exponent, "^")
		case '!':
			return NewLexerToken(pos, LogicalNot, "!")
		case '(':
			return NewLexerToken(pos, LeftParenthesis, "(")
		case ')':
			return NewLexerToken(pos, RightParenthesis, ")")
		case '{':
			return NewLexerToken(pos, LeftCurlyBrace, "{")
		case '}':
			return NewLexerToken(pos, RightCurlyBrace, "}")
		case '[':
			return NewLexerToken(pos, LeftBracket, "[")
		case ']':
			return NewLexerToken(pos, RightBracket, "]")
		}
		switch {
		case unicode.IsSpace(rune):
			continue
		case unicode.IsDigit(rune) || rune == '.':
			return l.ParseNumber(pos)
		case unicode.IsLetter(rune), rune == '_':
			tt, val := l.ParseIdentifier()
			return NewLexerToken(pos, tt, val)
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
	l.Backup()
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
			return
		}
	}
}
