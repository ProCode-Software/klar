package lexer

import (
	"io"
	"strings"
	"unicode"
)

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

func (l *Lexer) ReadOperator(r rune) (TokenType, string) {
	n, ok := opPrefixes[r] // n = length of longest operator - 1
	singleStr := string(r)
	if !ok {
		return Illegal, singleStr
	}
	for ; n > 0; n-- {
		next, isEOF := l.PeekN(n)
		if isEOF {
			continue
		}
		total := singleStr + string(next)
		if opTok, ok := OperatorMap[total]; ok {
			l.Reader.Discard(n)    // l.Reader.Read(make([]byte, n))
			l.Pos.Col += uint32(n) // All operators are ASCII
			return opTok, total
		}
	}
	return OperatorMap[singleStr], singleStr
}

func (l *Lexer) ReadShebang(pos Position) *Token {
	tok := l.ReadLineComment(pos)
	tok.Kind = Hashbang
	tok.Source = "#!" + tok.Source[2:] // "//"
	return tok
}

func (l *Lexer) ReadLineComment(pos Position) *Token {
	var leng uint32
	t := l.NewTokenizer(true)
	for r, b := range t.Tokenize {
		// Beginning // is already parsed
		if r == '\n' {
			break
		}
		b.WriteRune(r)
		leng++
	}
	return NewToken(pos, LineComment, "//"+t.String()).withAttrs(attrs{"length": leng})
}

func (l *Lexer) ReadBlockComment(pos Position) *Token {
	var (
		cmtLevel = 1
		last     rune
		leng     uint32
		t        = l.NewTokenizer(false)
	)
	for r, b := range t.Tokenize {
		leng++
		b.WriteRune(r)
		if last == '*' && r == '/' {
			if cmtLevel--; cmtLevel == 0 {
				break
			}
		} else if last == '/' && r == '*' {
			cmtLevel++
		}
		last = r
	}
	return NewToken(pos, BlockComment, t.String()).
		withAttrs(attrs{"unterm": t.EOF(), "end": l.Pos, "length": leng})
}

const (
	_ = iota
	ErrIntMisplacedSeparator
	ErrIntIncompatibleDigit
	ErrStrUnterminated
)

func (l *Lexer) ReadNumber(pos Position) *Token {
	var (
		format                   IntFormat
		errorType, errPos        int
		isExp, isDecimal, hasSep bool
		last                     rune
	)
	newError := func(code int, b *Builder) {
		errorType = code
		if b != nil {
			errPos = b.Len()
		}
	}
	l.Backup()
	t := l.NewTokenizer(true)
loop:
	for r, b := range t.Tokenize {
		lower := unicode.ToLower(r)
		if b.String() == "0" {
			switch lower {
			case 'x':
				format = NumberFormatHex
			case 'o':
				format = NumberFormatOctal
			case 'b':
				format = NumberFormatBinary
			default:
				format = NumberFormatDecimal
				break loop
			}
			b.WriteRune(r)
			last = r
			continue
		}
		switch lower {
		case 'e':
			if format == NumberFormatDecimal && !isExp {
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b)
					errPos--
				}
				b.WriteRune(r)
				isExp = true
				last = r
				continue
			}
			fallthrough
		case 'a', 'b', 'c', 'd', 'f':
			switch format {
			case NumberFormatHex:
				b.WriteRune(r)
			default:
				// Hex letter or e on other format
				newError(ErrIntIncompatibleDigit, b)
				// b.WriteRune(r)
			}
		case '+', '-':
			if !isExp {
				break loop
			}
			b.WriteRune(r)
		case '.':
			switch {
			case isDecimal:
				break loop
			case format != NumberFormatDecimal:
				newError(ErrIntIncompatibleDigit, b)
				fallthrough
			default:
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b)
					errPos--
				}
				if n, isEOF := l.BackupPeek(); isEOF || !IsDigit(rune(n)) {
					break loop
				}
				isDecimal = true
				b.WriteRune(r)
			}
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if last == '_' || (format == NumberFormatDecimal && !IsDigit(last)) {
				newError(ErrIntMisplacedSeparator, b)
			}
			hasSep = true
			b.WriteRune(r)
		default:
			switch {
			case !IsDigit(r):
				break loop
			case
				format == NumberFormatDecimal,
				format == NumberFormatHex,
				format == NumberFormatBinary && r <= '1',
				format == NumberFormatOctal && r <= '7':
				b.WriteRune(r)
			default:
				newError(ErrIntIncompatibleDigit, b)
				b.WriteRune(r)
			}
		}
		last = r
	}
	digit := t.String()
	// Last digit is separator
	if digit[len(digit)-1] == '_' {
		newError(ErrIntMisplacedSeparator, nil)
		errPos = len(digit) - 1
	}
	var flags NumberFlags
	if isExp {
		flags |= HasExponent | IsFloat
	}
	if isDecimal {
		flags |= IsFloat
	}
	if hasSep {
		flags |= HasSeparator
	}
	var err *NumberError
	if errorType != 0 {
		err = &NumberError{Code: errorType, Offset: uint32(errPos)}
	}
	return NewToken(pos, Numeric, digit).withAttrs(attrs{
		"params": NumberAttrs{
			Format: format,
			Flags:  flags,
			Error:  err,
		},
	})
}

type IntFormat uint8

const (
	NumberFormatDecimal IntFormat = iota
	NumberFormatHex
	NumberFormatOctal
	NumberFormatBinary
)

type NumberFlags uint8

const (
	IsFloat NumberFlags = 1 << iota
	HasSeparator
	HasExponent
)

type NumberAttrs struct {
	Format IntFormat
	Flags  NumberFlags
	Error  *NumberError
}

type NumberError struct {
	Code   int
	Offset uint32
}

func (l *Lexer) ReadIdentifier(start Position) *Token {
	var length uint32
	l.Backup()
	t := l.NewTokenizer(true)
	for r, b := range t.Tokenize {
		// Use unicode.IsDigit to allow digit in any language
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			length++
		} else {
			break
		}
	}
	id := t.String()
	if keyword, is := KeywordMap[id]; is {
		tok := NewToken(start, keyword, id)
		if keyword == Boolean {
			tok.setAttr("value", id == "true")
		}
		return tok
	}
	return NewToken(start, Identifier, id).withAttrs(attrs{"length": length})
}

type RegexAttrs struct {
	Flags        []rune
	Source       string
	Unterminated bool
	Multiline    bool
	SlashCount   int
}

func (l *Lexer) ReadRegex(startPos Position, slashN int) *Token {
	var (
		unterm                bool
		slashCt               int
		lastSlashEnd          = startPos.Col + 1 + uint32(slashN)
		hasNewline, isNewline bool
		end                   Position
		b                     strings.Builder
		leng                  uint32
	)
loop:
	for {
		r, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			unterm = true
			break loop
		}
		l.Pos.Col++
		leng++
		switch r {
		case '/':
			slashCt++
			if slashCt >= slashN {
				end = l.Pos
				b.WriteRune(r)
				break loop
			}
		case '\n':
			hasNewline, isNewline = true, true
			continue
		default:
			if isNewline && unicode.IsSpace(r) && l.Pos.Col-1 <= lastSlashEnd {
				continue
			}
		}
		b.WriteRune(r)
		isNewline = false
	}
	var prefix string
	if slashN > 0 {
		prefix = repeat('@', '/', slashN)
	} else {
		prefix = "/"
	}
	t := l.NewTokenizer(true)
	var flags []rune
	for r := range t.Tokenize {
		if isASCIILetter(r) {
			b.WriteRune(r) // Append to full source
			end = l.Pos
			flags = append(flags, r)
		} else {
			break
		}
	}
	str := prefix + b.String()
	return NewToken(startPos, Regex, str).withAttrs(attrs{
		"end":    end,
		"length": leng,
		"params": RegexAttrs{
			Source:       str,
			Multiline:    hasNewline,
			Flags:        flags,
			Unterminated: unterm,
			SlashCount:   slashN,
		},
	})
}
