package printer

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type PrintFlags uint8

const (
	PrettyPrint PrintFlags = 1 << iota
	SingleLine
)

func space(n uint32) []byte {
	if n > 10000000 && n >= ^uint32(0)-100 {
		panic("overflow of n")
	}
	arr := make([]byte, n)
	for i := range n {
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
	if f&PrettyPrint != 0 {
		return prettyPrintTokens(tokens, f)
	}
	return defaultPrintTokens(tokens, f)
}

func defaultPrintTokens(tokens []lexer.Token, flags PrintFlags) []byte {
	b := make([]byte, 0, len(tokens)*2)
	var lastLine, lastCol uint32 = tokens[0].Line, tokens[0].Col
	for _, tok := range tokens {
		if flags & SingleLine != 0 {
			b = append(b, ' ')
		} else {
			
			b = append(b, '\n')
		}
		for currLine := lastLine; currLine < tok.Line; currLine++ {
			if flags & SingleLine != 0 {
				b = append(b, ' ')
			} else {
				b = append(b, '\n')
			}
			lastCol = 1
		}
		b = append(b, space(tok.Col-lastCol)...)
		b = append(b, []byte(tok.Source)...)
		tokRange := ranges.FromToken(tok).End
		lastLine, lastCol = tokRange.Line, tokRange.Col
	}
	return b[len(b):]
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
		if addSpaceBefore(kind) {
			b = append(b, ' ')
		}
		for currLine := lastLine; currLine < tok.Line; currLine++ {
			b = append(b, '\n')
			lastCol = 1
		}
		b = append(b, space(tok.Col-lastCol)...)
		b = append(b, []byte(tok.Source)...)
		tokRange := ranges.FromToken(tok).End
		lastLine, lastCol = tokRange.Line, tokRange.Col
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
	spaceBefore[lexer.RightCurlyBrace] = struct{}{}
}
