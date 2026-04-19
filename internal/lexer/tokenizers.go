package lexer

import "unicode"

// ReadIdentifier reads an identifier or keyword. Underscores and
// letters and digits in any language are allowed in identifiers.
// first must not be a digit.
func (l *Lexer) ReadIdentifier(start Position, first rune) *Token {
	var length uint32 = 1
	t := l.NewTokenizer(true)
	t.Builder.WriteRune(first)
	for r, b := range t.Tokenize {
		// Use unicode.IsDigit to allow digit in any language
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			length++
			continue
		}
		break
	}
	id := t.String()
	// Check for keywords
	if keyword, ok := KeywordMap[id]; ok {
		tok := NewToken(start, keyword, id)
		if keyword == Boolean {
			tok.setAttr("value", id == "true")
		}
		return tok
	}
	return NewToken(start, Identifier, id).withAttrs(attrs{"length": length})
}

// ReadOperator reads a single operator token. None of the operators read
// start with a letter. See [Lexer.ReadIdentifier] for reading keywords.
func (l *Lexer) ReadOperator(r rune) (TokenType, string) {
	n, ok := opPrefixes[r] // n = length of longest operator - 1
	singleStr := string(r)
	if !ok {
		return Illegal, singleStr
	}
	// Backup each time
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
	tok.Source = "#!" + tok.Source[2:] // Replace "//"
	return tok
}

func (l *Lexer) ReadLineComment(pos Position) *Token {
	var leng uint32 = 2
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
		t               = l.NewTokenizer(false)
		leng     uint32 = 2
		cmtLevel        = 1
		last     rune
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
	return NewToken(pos, BlockComment, "/*"+t.String()).
		withAttrs(attrs{"unterm": t.EOF(), "end": l.Pos, "length": leng})
}

type RegexAttrs struct {
	Flags        []byte
	Source       string // Actual expression contents
	Unterminated bool
	Multiline    bool
	DoubleSlash  bool
}

// TODO: check on this after RFC is approved
func (l *Lexer) ReadRegex(startPos Position) *Token {
	var (
		slashEnd              = startPos.Col + 1
		hasNewline, isNewline bool
		leng                  uint32
		t                     = l.NewTokenizer(false)
	)
	// Regex contents
	// ============
	const prefix = "#/"
	t.Builder.WriteString(prefix)
loop:
	for r, b := range t.Tokenize {
		switch r {
		case '/':
			b.WriteRune(r)
			leng++
			break loop
		case '\n':
			hasNewline, isNewline = true, true
			continue
		default:
			// Trim whitespace at the beginning of each line, similar to strings
			if isNewline && unicode.IsSpace(r) && l.Pos.Col-1 <= slashEnd {
				continue
			}
			b.WriteRune(r)
			leng++
			isNewline = false
		}
	}
	unterm := t.EOF()
	srcEnd := t.Builder.Len() - 1

	// Flags
	// =======
	t.ResetKeepBuilder(true)
	var flags []byte
	for r, b := range t.Tokenize {
		if !IsASCIILetter(r) {
			break
		}
		c := byte(r)
		b.WriteByte(c) // Append to full source
		flags = append(flags, c)
		leng++
	}

	str := t.String()
	return NewToken(startPos, Regex, str).withAttrs(attrs{
		"end":    t.EndPos(),
		"length": leng,
		"params": RegexAttrs{
			Source:       str[len(prefix):srcEnd],
			Multiline:    hasNewline,
			Flags:        flags,
			Unterminated: unterm,
			DoubleSlash:  false,
		},
	})
}
