package lexer

import (
	"fmt"
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

func (l *Lexer) ParseOperator(r rune) (TokenType, string) {
	n, ok := opPrefixes[r]
	singleStr := string(r)
	if !ok {
		return Illegal, singleStr
	}
	for ; n > 0; n-- {
		next, isEOF := l.PeekN(n)
		if isEOF {
			continue
		}
		total := string(r) + string(next)
		if opTok, ok := OperatorMap[total]; ok {
			// Check if next byte is an indentifier if operator ends in ident
			if IsIdent(rune(total[n])) && l.checkIfIdentNext(n) {
				continue
			}
			l.Reader.Discard(n) // l.Reader.Read(make([]byte, n))
			l.Pos.Col += uint32(n)
			return opTok, total
		}
	}
	return OperatorMap[singleStr], singleStr
}

func (l *Lexer) checkIfIdentNext(n int) bool {
	if next, isEOF := l.PeekN(n + 1); !isEOF {
		first := rune(next[n])
		if IsIdent(first) || unicode.IsDigit(first) {
			return true
		}
	}
	return false
}

func (l *Lexer) ParseShebang(pos Position) *Token {
	tok := l.ParseLineComment(pos)
	tok.Kind = Hashbang
	tok.Source = "#!" + tok.Source[2:]
	return tok
}

func (l *Lexer) ParseLineComment(pos Position) *Token {
	var shouldStop bool
	cmt := l.TokenizeFunc(func(r rune, b *Builder) bool {
		if shouldStop {
			return false
		}
		// Beginning // is already parsed
		if r == '\n' {
			shouldStop = true
			return true
		}
		b.WriteRune(r)
		return true
	})
	return NewToken(pos, LineComment, "//"+cmt)
}

func (l *Lexer) ParseBlockComment(pos Position) *Token {
	var (
		endPos   Position
		cmtLevel = 1
		unterm   bool
		b        Builder
		last     rune
	)
	b.WriteString("/*")
loop:
	for {
		r, _, err := l.Reader.ReadRune()
		l.Pos.Col++
		if handleReadError(err) {
			unterm = true
			endPos = l.Pos
			break loop
		}
		switch {
		case r == '/' && last == '*':
			if b.Len() > 2 {
				cmtLevel--
			}
			if cmtLevel == 0 {
				b.WriteRune(r)
				endPos = l.nextCol()
				break loop
			}
		case r == '*' && last == '/':
			cmtLevel++
		case r == '\n':
			l.ResetPosition()
		}
		last = r
		b.WriteRune(r)
	}

	return NewToken(pos, BlockComment, b.String()).
		SetAttribute("unterm", unterm).SetAttribute("end", endPos)
}

const (
	_ = iota
	ErrIntMisplacedSeparator
	ErrIntIncompatibleDigit
	ErrStrUnterminated
)

func (l *Lexer) ParseNumber(pos Position) *Token {
	var (
		format, errorType, errPos   int
		isExp, isIllegal, isDecimal bool
		last                        rune
	)
	newError := func(code int, b *Builder) {
		errorType = code
		if b != nil {
			errPos = b.Len()
		}
		isIllegal = true
	}
	digit := l.BackupTokenizeFunc(func(r rune, b *Builder) bool {
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
				return false
			}
			b.WriteRune(r)
			last = r
			return true
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
				return true
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
				return false
			}
			b.WriteRune(r)
		case '.':
			switch {
			case isDecimal:
				return false
			case b.Len() == 0: // .3
				format = NumberFormatDecimal
				b.WriteRune(r)
				isDecimal = true
			case format != NumberFormatDecimal:
				newError(ErrIntIncompatibleDigit, b)
				fallthrough
			default:
				if last == '_' {
					newError(ErrIntMisplacedSeparator, b)
					errPos--
				}
				n, isEOF := l.BackupPeek()
				if n == '.' {
					return false // 1...10
				}
				// Trailing decimal point at EOF
				if isEOF || IsDigit(rune(n)) || n == 'e' || n == 'E' || !IsIdent(r) {
					isDecimal = true
					b.WriteRune(r)
				} else {
					return false
				}
			}
		case '_':
			// Underscore separators: no consecutive, must be in between digits
			if last == '_' || (format == NumberFormatDecimal && !IsDigit(last)) {
				newError(ErrIntMisplacedSeparator, b)
			}
			b.WriteRune(r)
		default:
			switch {
			case !IsDigit(r):
				return false
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
		return true
	})
	// Last digit is separator
	if digit[len(digit)-1] == '_' {
		newError(ErrIntMisplacedSeparator, nil)
		errPos = len(digit) - 1
	}
	return NewToken(pos, Numeric, digit).SetAttribute("params", NumberAttrs{
		Format:  format,
		Float:   isDecimal || isExp,
		Invalid: isIllegal,
		ErrPos:  errPos,
		Error:   errorType,
	})
}

type NumberAttrs struct {
	Format        int
	Float         bool
	Invalid       bool
	Error, ErrPos int
}

func (l *Lexer) ParseIdentifier() (TokenType, string) {
	id := l.BackupTokenizeFunc(func(r rune, b *Builder) bool {
		// Use unicode.IsDigit to allow digit in any language
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			return true
		}
		return false
	})
	if keyword, is := KeywordMap[id]; is {
		return keyword, id
	}
	return Identifier, id
}

type RegexAttrs struct {
	Flags        []rune
	Source       string
	Unterminated bool
	Multiline    bool
	SlashCount   int
}

func (l *Lexer) ParseRegex(startPos Position, slashN int) *Token {
	var (
		unterm                bool
		slashCt               int
		lastSlashEnd          = startPos.Col + 1 + uint32(slashN)
		hasNewline, isNewline bool
		end                   Position
		b                     strings.Builder
	)
loop:
	for {
		r, _, err := l.Reader.ReadRune()
		if handleReadError(err) {
			unterm = true
			break loop
		}
		l.Pos.Col++
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
	var prefix []rune
	if slashN > 0 {
		prefix = make([]rune, slashN+1)
		prefix[0] = '@'
		for i := range slashN {
			prefix[i+1] = '/'
		}
	} else {
		prefix = []rune{'/'}
	}
	fmt.Println(end)
	flagStr := l.TokenizeFunc(func(r rune, _ *Builder) bool {
		if isASCIILetter(r) {
			b.WriteRune(r)
			end = l.Pos
			return true
		}
		return false
	})
	str := string(prefix) + b.String()
	fmt.Println(end, len(str))
	return NewToken(startPos, Regex, str).
		SetAttribute("end", end).
		SetAttribute("params", RegexAttrs{
			Source:       "",
			Multiline:    hasNewline,
			Flags:        []rune(flagStr),
			Unterminated: unterm,
			SlashCount:   slashN,
		})
}
