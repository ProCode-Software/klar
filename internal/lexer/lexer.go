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
			return NewLexerToken(pos, EOF, "")
		}
		l.Pos.Col++

		switch r {
		case '\n':
			l.ResetPosition()
			return NewLexerToken(pos, Newline, "\n")
		case '"', '\'', '`':
			return l.ParseString(pos)
		case '!', '+', ':', '-', '&', '|', '=', '>', '<', '/':
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
				src, endPos := l.ParseBlockComment()
				if !l.IncludeComments {
					continue
				}
				return NewLexerToken(pos, BlockComment, src).SetAttribute("end", endPos)
			}
			// Keep going if it's a dot
			if !(tt == Illegal && val == ".") {
				return NewLexerToken(pos, tt, val)
			}
		// Single-character operators
		case '@':
			return NewLexerToken(pos, At, "@")
		case '*':
			return NewLexerToken(pos, Asterisk, "*")
		case '%':
			return NewLexerToken(pos, Percent, "%")
		case '^':
			return NewLexerToken(pos, Caret, "^")
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
		case ',':
			return NewLexerToken(pos, Comma, ",")
		case '?':
			return NewLexerToken(pos, Question, "?")
		case '#':
			next, err := l.Reader.Peek(1)
			if handleReadError(err) || next[0] != '{' {
				return NewLexerToken(pos, Illegal, "#")
			}
			l.Reader.ReadRune()
			l.Pos.Col++
			return NewLexerToken(pos, HashLeftCurlyBrace, "#{")
		case '.':
			if err := l.Reader.UnreadRune(); err != nil {
				panic(err)
			}
			next, err := l.Reader.Peek(2)
			l.Reader.ReadRune()
			if handleReadError(err) {
				return NewLexerToken(pos, Dot, ".")
			}
			if unicode.IsDigit(rune(next[1])) {
				return l.ParseNumber(pos)
			}
			if next[1] == '.' {
				l.Reader.ReadRune()
				l.Pos.Col++
				next, err = l.Reader.Peek(1)
				if handleReadError(err) || next[0] != '.' {
					return NewLexerToken(pos, Illegal, "..")
				} else {
					l.Reader.ReadRune()
					l.Pos.Col++
					return NewLexerToken(pos, Spread, "...")
				}
			}
			return NewLexerToken(pos, Dot, ".")
		}
		switch {
		case unicode.IsSpace(r):
			continue
		case unicode.IsDigit(r):
			return l.ParseNumber(pos)
		case unicode.IsLetter(r), r == '_':
			tt, val := l.ParseIdentifier()
			return NewLexerToken(pos, tt, val)
		default:
			return NewLexerToken(pos, Illegal, string(r))
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
