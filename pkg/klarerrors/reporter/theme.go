package reporter

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

type ColorPalette struct {
	TokenColors   map[lexer.TokenType]string
	TypeColor     string
	FunctionColor string
	EscapeColor   string
}

func DefaultColorPalette() *ColorPalette {
	if defaultColors == nil {
		defaultColors = makeDefaultColors()
	}
	return defaultColors
}

// TODO: other palettes: frost, github

type CharacterSet struct {
	HighlightSingle rune
	HighlightMulti  rune
	BoxTL           rune
	BoxT            rune
	BoxL            rune
	HighlightLine   rune
	SkipLine        rune
	SkipLineL       rune

	HighlightMultilineTL     rune
	HighlightMultilineBL     rune
	HighlightMultilineOffset rune
	HighlightMultilineSkip   rune

	ArrowT  rune
	ArrowBL rune
	ArrowH  rune
	ArrowV  rune
}

func DefaultCharacterSet() *CharacterSet {
	return &CharacterSet{
		HighlightSingle: '^',
		HighlightMulti:  '━', // '─'
		BoxTL:           '┌',
		BoxT:            '─',
		BoxL:            '│',
		HighlightLine:   '·',
		SkipLine:        '-', // ╌
		SkipLineL:       '┤',

		HighlightMultilineTL:     '╭',
		HighlightMultilineBL:     '╰',
		HighlightMultilineOffset: '·',
		HighlightMultilineSkip:   '├',

		ArrowT:  '┬',
		ArrowBL: '╰',
		ArrowH:  '─',
		ArrowV:  '│',
	}
}

func ASCIICharacterSet() *CharacterSet {
	return &CharacterSet{
		HighlightSingle: '^',
		HighlightMulti:  '~',
		BoxTL:           '|',
		BoxT:            '-',
		BoxL:            '|',
		HighlightLine:   '.',
		SkipLine:        '=',
		SkipLineL:       '+',

		HighlightMultilineTL:     'r',
		HighlightMultilineBL:     '\'',
		HighlightMultilineOffset: '.',
		HighlightMultilineSkip:   '=',

		ArrowT:  ',',
		ArrowBL: '`',
		ArrowH:  '-',
		ArrowV:  '|',
	}
}
