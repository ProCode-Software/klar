package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type StringAttrs struct {
	Fragments    []StringFragment // Between newlines and escapes (newline at end)
	QuoteStyle   rune
	Unterminated bool
}

// String escapes
// =========

// Note: Interpolations are [StringEscape].
type StringEscape struct {
	Pos          Position
	Type         EscapeType
	Value        string
	Interpolated *[]Token
	Error        *EscapeError
}

type EscapeType int

const (
	_ EscapeType = iota
	EscCharacter
	EscHex
	EscUnicode
	EscInterpolation
)

type EscapeError struct {
	Code EscapeErrorCode
	Pos  Position
}

type EscapeErrorCode int

const (
	_ EscapeErrorCode = iota
	ErrEscapeTooShort
	ErrUnicodeEscapeTooLong
	ErrEscapeUnterm
	ErrEscapeExpHex
	ErrCharEscapeUnknown
)

// String fragments
// ============

type StringFragment interface {
	StringFrag()
	String() string
}

type TextFragment struct {
	Source string
	// Number of spaces before this fragment begins if this is the
	// first fragment on the line
	LineOffset uint32
}

func (TextFragment) StringFrag()      {}
func (StringEscape) StringFrag()      {}
func (t TextFragment) String() string { return t.Source }
func (e StringEscape) String() string { return e.Value }

func (l *Lexer) readUnicodeEsc(delim rune) StringEscape {
	var (
		err     *EscapeError
		last    rune
		invalid = func(code EscapeErrorCode) { err = &EscapeError{code, l.Pos} }
		t       = l.NewTokenizer(true)
	)
loop:
	for r, b := range t.Tokenize {
		l := b.Len()
		switch {
		case l > 0 && last == '}':
			break loop
		case l < 2 && r == '}':
			invalid(ErrEscapeTooShort)
			b.WriteRune(r)
		case r == '}', IsHex(r), l == 0 && r == '{':
			b.WriteRune(r)
		case l > 6 && r != '}':
			invalid(ErrUnicodeEscapeTooLong)
			break loop
		case r == delim:
			invalid(ErrEscapeUnterm)
			break loop
		default:
			invalid(ErrEscapeExpHex)
			break loop
		}
		last = r
	}
	if t.EOF() && last != '}' {
		invalid(ErrEscapeUnterm)
	}
	return StringEscape{
		Type:  EscUnicode,
		Value: `\u` + t.String(),
		Error: err,
	}
}

func (l *Lexer) readHexEsc(delim rune) StringEscape {
	var (
		err     *EscapeError
		invalid = func() { err = &EscapeError{ErrEscapeExpHex, l.Pos} }
		t       = l.NewTokenizer(true)
	)
loop:
	for r, b := range t.Tokenize {
		switch {
		case b.Len() == 2:
			break loop
		case IsHex(r):
			b.WriteRune(r)
		case r == delim:
			invalid()
			break loop
		default:
			invalid()
			b.WriteRune(r)
		}
	}
	esc := t.String()
	if t.EOF() && len(esc) < 2 {
		invalid()
	}
	return StringEscape{
		Type:  EscHex,
		Value: `\x` + esc,
		Error: err,
	}
}

// '{' already parsed
func (l *Lexer) readStrInterp() StringEscape {
	var (
		err     *EscapeError
		tokens  []Token
		braceCt = 1
		b       strings.Builder
	)
	b.WriteRune('{')
loop:
	for {
		// TODO: rewrite this part. this sucks
		if next, eof := l.PeekN(2); !eof {
			if r, _ := utf8.DecodeRune(next); unicode.IsSpace(r) {
				b.WriteRune(r)
			}
		}
		t := l.Tokenize()
		switch t.Kind {
		case EOF:
			err = &EscapeError{ErrEscapeUnterm, l.Pos}
			break loop
		case LeftCurlyBrace, HashLeftCurlyBrace:
			braceCt++
		case RightCurlyBrace:
			braceCt--
			if braceCt == 0 {
				if len(tokens) == 0 {
					err = &EscapeError{ErrEscapeTooShort, l.Pos}
				} else {
					tokens[len(tokens)-1].setAttr("end", l.Pos)
				}
				b.WriteString(t.Source)
				break loop
			}
		}
		tokens = append(tokens, *t)
		b.WriteString(t.Source)
	}
	return StringEscape{
		Type:         EscInterpolation,
		Interpolated: &tokens,
		Value:        b.String(),
		Error:        err,
	}
}

/*
TODO: wrap strings (multiline src, joined to single line); trim newline around `
Beginning is already parsed

There are three types of strings in Klar:

	Double-quote "": Interpolation, escapes
	Single-quote '': Raw, no escapes
	Backtick ``: Interpolation, escapes, multiline
*/
// TODO: leng doesn't include escapes
func (l *Lexer) ReadString(pos Position, delim rune) *Token {
	var (
		fragStart                   int
		leng, firstOffset           uint32
		isEscape, isNewline, unterm bool
		escStart                    Position
		frags                       []StringFragment
		t                           = l.NewTokenizer(false)
	)
	newEscape := func(e StringEscape) {
		if e.Type == EscInterpolation {
			t.Builder.WriteString(e.Value)
		} else {
			t.Builder.WriteString(e.Value[1:]) // Character after \
		}
		e.Pos = escStart
		frags = append(frags, e)
		isEscape, fragStart = false, t.Builder.Len()
	}
	charEscape := func(c rune, errCode EscapeErrorCode) StringEscape {
		esc := StringEscape{
			Type:  EscCharacter,
			Value: `\` + string(c),
		}
		if errCode != 0 {
			esc.Error = &EscapeError{errCode, l.Pos}
		}
		return esc
	}
	endFrag := func() { // End a fragment before '\' in escape or after '\n'
		frags = append(frags, TextFragment{
			Source:     t.Builder.String()[fragStart:],
			LineOffset: firstOffset,
		})
		firstOffset = 0
	}
loop:
	for r, b := range t.Tokenize {
		leng++
		if isEscape {
			switch r {
			case '\\', '{', 'b', 'e', 'f', 'n', 'r', 't', delim:
				newEscape(charEscape(r, 0))
			case 'x':
				newEscape(l.readHexEsc(delim))
			case 'u':
				newEscape(l.readUnicodeEsc(delim))
			default:
				newEscape(charEscape(r, ErrCharEscapeUnknown))
			}
			isNewline = false
			continue loop
		}
		switch r {
		case delim:
			endFrag()
			b.WriteRune(r)
			break loop
		case '\\':
			endFrag()
			isEscape, escStart = true, l.prevCol()
		case '{':
			if delim != '\'' { // " or `
				escStart = l.prevCol()
				newEscape(l.readStrInterp())
				continue loop
			}
		case '\n':
			b.WriteRune(r)
			if delim != '`' { // " or '
				// Invalid newline, just stop parsing
				unterm = true
				endFrag()
				break loop
			}
			isNewline = true
			endFrag()
			fragStart = b.Len()
			continue loop
		default:
			// Strip leading spaces from backtick string
			if isNewline && unicode.IsSpace(r) && l.Pos.Col-1 <= pos.Col {
				firstOffset++
				continue loop
			}
		}
		b.WriteRune(r)
		isNewline = false
	}
	if t.EOF() {
		unterm = true
		endFrag()
	}
	return NewToken(pos, String, string(delim)+t.String()).withAttrs(attrs{
		"end":    t.EndPos(),
		"length": leng,
		"params": StringAttrs{
			QuoteStyle:   delim,
			Unterminated: unterm,
			Fragments:    frags,
		},
	})
}
