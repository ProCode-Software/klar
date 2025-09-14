package printer

import (
	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Options passed to [PrintTokens]
type PrintFlags uint8

const (
	// Perform simple formatting such as adding spaces around operands and curly braces.
	PrettyPrint PrintFlags = 1 << iota
	// Print tokens on a single line
	SingleLine
)

func space(n uint32) []byte {
	if n > 10000000 && n >= ^uint32(0)-100 {
		panic("overflow of n")
	}
	cl := uint32(char.Length)
	if n <= cl {
		return char.Spaces[:n]
	}
	// More than 32 spaces
	arr := make([]byte, n)
	copy(arr, char.Spaces)
	for i := cl; i < n; i++ {
		arr[i] = ' '
	}
	return arr
}

// PrintTokens prints tokens preserving the position
func PrintTokens(tokens []lexer.Token, flags ...PrintFlags) []byte {
	var f PrintFlags
	for _, flag := range flags {
		f |= flag
	}
	if (f & PrettyPrint) != 0 {
		return prettyPrintTokens(tokens, f)
	}
	return defaultPrintTokens(tokens, f)
}

func defaultPrintTokens(tokens []lexer.Token, flags PrintFlags) []byte {
	b := make([]byte, 0, len(tokens)*2)
	lastLine, lastCol := tokens[0].Line, tokens[0].Col
	for _, tok := range tokens {
		if lastLine < tok.Line {
			if (flags & SingleLine) != 0 {
				b = append(b, ' ')
			} else {
				for currLine := lastLine; currLine < tok.Line; currLine++ {
					b = append(b, '\n')
				}
				lastCol = 1
			}
		}
		b = append(b, space(tok.Col-lastCol)...)
		b = append(b, []byte(tok.Source)...)
		tokRange := ranges.FromToken(tok).End
		lastLine, lastCol = tokRange.Line, tokRange.Col
	}
	return b[:len(b):len(b)]
}

var (
	spaceAfter  = map[lexer.TokenType]struct{}{}
	spaceAround = map[lexer.TokenType]struct{}{}
	spaceBefore = map[lexer.TokenType]struct{}{}
)

// PrettyPrintTokens prints tokens similar to [PrintTokens], but does simple
// formatting such as adding spaces around operands and curly braces.
func prettyPrintTokens(tokens []lexer.Token, flags PrintFlags) []byte {
	b := make([]byte, 0, len(tokens)*2)
	lastLine := tokens[0].Line
	addSpaceBefore := func(t lexer.TokenType) bool {
		_, ok := spaceBefore[t]
		_, ok2 := spaceAround[t]
		return ok || ok2
	}
	addSpaceAfter := func(t lexer.TokenType) bool {
		_, ok := spaceAfter[t]
		_, ok2 := spaceAround[t]
		return ok || ok2
	}
	for _, tok := range tokens {
		kind := tok.Kind
		// If printing multiline
		if flags&SingleLine == 0 && lastLine < tok.Line {
			for currLine := lastLine; currLine < tok.Line; currLine++ {
				b = append(b, '\n')
			}
			if addSpaceBefore(kind) {
				b = append(b, char.Spaces[:4]...) // 4 spaces
			}
		} else if addSpaceBefore(kind) {
			b = append(b, ' ')
		}
		b = append(b, []byte(tok.Source)...)
		if addSpaceAfter(kind) {
			b = append(b, ' ')
		}
		lastLine = ranges.FromToken(tok).End.Line
	}
	return b[:len(b):len(b)]
}

// Pretty print rules
func init() {
	for _, tt := range lexer.KeywordMap {
		spaceAfter[tt] = struct{}{}
	}
	for _, tt := range lexer.OperatorMap {
		spaceAround[tt] = struct{}{}
	}
	delete(spaceAround, lexer.Ellipsis)
	delete(spaceAround, lexer.DotDotLessThan)
	delete(spaceAround, lexer.At)
	delete(spaceAround, lexer.Colon)
	delete(spaceAround, lexer.Comma)
	delete(spaceAround, lexer.Hash)
	delete(spaceAround, lexer.Dot)
	delete(spaceAround, lexer.LeftBracket)
	delete(spaceAround, lexer.RightBracket)
	delete(spaceAround, lexer.LeftParenthesis)
	delete(spaceAround, lexer.RightParenthesis)
	delete(spaceAround, lexer.LeftCurlyBrace)
	delete(spaceAround, lexer.RightCurlyBrace)
	delete(spaceAround, lexer.HashLeftCurlyBrace)
	delete(spaceAround, lexer.LineComment)
	delete(spaceAround, lexer.BlockComment)
	delete(spaceAround, lexer.Hashbang)
	delete(spaceAround, lexer.NotCan)
	delete(spaceAround, lexer.Backslash)
	delete(spaceAround, lexer.Not)
	delete(spaceAround, lexer.PlusPlus)
	delete(spaceAround, lexer.MinusMinus)

	delete(spaceAfter, lexer.Boolean)
	delete(spaceAfter, lexer.Nil)
	delete(spaceAfter, lexer.Underscore)

	spaceAround[lexer.In] = struct{}{}
	spaceAround[lexer.And] = struct{}{}
	spaceAround[lexer.Or] = struct{}{}
	spaceAfter[lexer.LeftCurlyBrace] = struct{}{}
	spaceAfter[lexer.HashLeftCurlyBrace] = struct{}{}
	spaceAfter[lexer.Comma] = struct{}{}
	spaceAfter[lexer.Colon] = struct{}{}
	spaceBefore[lexer.RightCurlyBrace] = struct{}{}
}
