package lexer

import (
	"unicode"
)

type EscapeType int

const (
	_ EscapeType = iota
	EscCharacter
	EscHex
	EscUnicode
	EscInterpolation
)

type EscapeError int

const (
	_ EscapeError = iota
	ErrEscapeTooShort
	ErrEscapeTooLong
	ErrEscapeUnterm
	ErrEscapeExpHex
	ErrEscapeUnknown
)

type StringAttrs struct {
	Fragments    []StringFragment // Between newlines and escapes (newline at end)
	QuoteStyle   rune
	QuoteCount   int // > 0 if @ was used
	Unterminated bool
}

type StringEscape struct {
	Offset        int
	Pos           Position
	Type          EscapeType
	Value         string
	Interpolated  *[]Token
	Invalid       EscapeError
	ErrorPosition *Position
}

type StringFragment interface {
	frag()
}

type TextFragment struct {
	Source string
}

func (TextFragment) frag()    {}
func (TextFragment) ASTFrag() {}
func (StringEscape) frag()    {}

func isHex(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
}

func (l *Lexer) readUnicodeEsc(delim rune) StringEscape {
	var (
		err    EscapeError
		errPos Position
		last   rune
	)
	invalid := func(reason EscapeError) {
		err, errPos = reason, l.Pos
	}
	esc := l.TokenizeEOFFunc(func(r rune, b *Builder) bool {
		l := b.Len()
		switch {
		case l > 0 && last == '}':
			return false
		case l < 2 && r == '}':
			invalid(ErrEscapeTooShort)
			b.WriteRune(r)
		case r == '}', isHex(r), l == 0 && r == '{':
			b.WriteRune(r)
		case l > 6 && r != '}':
			invalid(ErrEscapeTooLong)
			return false
		case r == delim:
			invalid(ErrEscapeUnterm)
			return false
		default:
			invalid(ErrEscapeExpHex)
			return false
		}
		last = r
		return true
	}, func() {
		if last != '}' {
			invalid(ErrEscapeUnterm)
		}
	})

	return StringEscape{
		Type:          EscUnicode,
		Value:         `\u` + esc,
		Invalid:       err,
		ErrorPosition: &errPos,
	}
}

func (l *Lexer) readHexEsc(delim rune) StringEscape {
	var (
		err     EscapeError
		errPos  Position
		unterm  bool
		invalid = func() { err, errPos = ErrEscapeExpHex, l.Pos }
	)
	esc := l.TokenizeEOFFunc(func(r rune, b *Builder) bool {
		l := b.Len()
		switch {
		case l == 2:
			return false
		case isHex(r):
			b.WriteRune(r)
		case r == delim:
			invalid()
			return false
		default:
			invalid()
			b.WriteRune(r)
		}
		return true
	}, func() { unterm = true })

	if unterm && len(esc) < 2 {
		invalid()
	}

	return StringEscape{
		Type:          EscHex,
		Value:         `\x` + esc,
		Invalid:       err,
		ErrorPosition: &errPos,
	}
}

// '{' already parsed
func (l *Lexer) readStrInterp() StringEscape {
	var (
		err     EscapeError
		errPos  Position
		tokens  []Token
		braceCt = 1
		b       Builder
	)
loop:
	for {
		t := l.Tokenize()
		switch t.Kind {
		case EOF:
			err, errPos = ErrEscapeUnterm, l.Pos
			break loop
		case LeftCurlyBrace, HashLeftCurlyBrace:
			braceCt++
		case RightCurlyBrace:
			braceCt--
			if braceCt == 0 {
				if len(tokens) == 0 {
					err, errPos = ErrEscapeTooShort, l.Pos
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
		Type:          EscInterpolation,
		Interpolated:  &tokens,
		Value:         "{" + b.String(),
		Invalid:       err,
		ErrorPosition: &errPos,
	}
}

/*
Beginning is already parsed

There are three types of strings in Klar:

	Double-quote "": Interpolation, escapes
	Single-quote '': Raw, no escapes
	Backtick ``: Interpolation, escapes, multiline
*/
func (l *Lexer) ReadString(pos Position, delim rune, quoteN int) *Token {
	var (
		currQuoteN, fragStart       int
		leng                        uint32
		b                           Builder
		isEscape, isNewline, unterm bool
		escapes                     map[Position]StringEscape
		end, escStart               Position
		frags                       []StringFragment
		lastQuoteEnd                = pos.Col + 1 + uint32(quoteN)
	)
	newEscape := func(e StringEscape) {
		if escapes == nil {
			escapes = make(map[Position]StringEscape)
		}
		b.WriteString(e.Value[1:]) // Character after \
		if e.Invalid > 0 {
			p := l.Pos
			e.ErrorPosition = &p
		}
		e.Offset, e.Pos = fragStart, escStart
		frags = append(frags, e)
		isEscape, fragStart = false, b.Len()
	}
	charEscape := func(c rune, err EscapeError) StringEscape {
		return StringEscape{Type: EscCharacter, Value: `\` + string(c), Invalid: err}
	}
	endFrag := func() { // End a fragment before '\' in escape or after '\n'
		frags = append(frags, TextFragment{b.String()[fragStart:]})
	}
loop:
	for {
		r, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			unterm = true
			break loop
		}
		l.Pos.Col++
		leng++
		if r != delim {
			currQuoteN = 0
		}
		if isEscape {
			switch r {
			case '\\', '{', 'b', 'e', 'f', 'n', 'r', 't', delim:
				newEscape(charEscape(r, 0))
			case 'x':
				newEscape(l.readHexEsc(delim))
			case 'u':
				newEscape(l.readUnicodeEsc(delim))
			default:
				newEscape(charEscape(r, ErrEscapeUnknown))
			}
			isNewline = false
			continue loop
		}
		switch r {
		case delim:
			if currQuoteN++; currQuoteN >= quoteN {
				// Cut existing quotes out of fragment
				frag := TextFragment{b.String()[fragStart:]}
				frag.Source = frag.Source[:len(frag.Source)-currQuoteN+1]
				frags = append(frags, frag)

				b.WriteRune(r)
				end = l.Pos
				break loop
			}
		case '\\':
			if quoteN == 0 { // No escape in @... string
				endFrag()
				isEscape, escStart = true, l.prevCol()
			}
		case '{':
			if delim != '\'' { // " or `
				escStart = l.prevCol()
				newEscape(l.readStrInterp())
				continue loop
			}
		case '\n':
			l.ResetPosition()
			b.WriteRune(r)
			if delim != '`' { // " or '
				// Invalid newline, just stop parsing
				unterm = true
				break loop
			}
			isNewline = true
			endFrag()
			fragStart = b.Len()
			continue loop
		default:
			// Strip leading spaces from backtick string
			if isNewline && unicode.IsSpace(r) && l.Pos.Col-1 <= lastQuoteEnd {
				continue loop
			}
		}
		b.WriteRune(r)
		isNewline = false
	}
	var prefix string
	if quoteN > 0 {
		prefix = repeat('@', delim, quoteN)
	} else {
		prefix = string(delim)
	}
	return NewToken(pos, String, string(prefix)+b.String()).withAttrs(attrs{
		"end":    end,
		"length": leng,
		"params": StringAttrs{
			QuoteStyle:   delim,
			Unterminated: unterm,
			QuoteCount:   quoteN,
			Fragments:    frags,
		},
	})
}
