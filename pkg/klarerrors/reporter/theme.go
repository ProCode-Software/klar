package reporter

import (
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/lexer"
)

type ColorPalette struct {
	TokenColors  map[lexer.TokenType]string
	Type         string // The color of type names
	Function     string // The color of function names
	BuiltinFunc  string // The color of built-in function names
	StringEscape string // The color of string escapes

	Divider  string // The color of error dividers
	Box      string // The color of the container and line numbers
	FileName string // The color of the file name in the header
	FilePos  string // The color of the line and column numbers in the header

	ErrorColor   string // The color of error highlights
	WarningColor string // The color of warning highlights
	HintColor    string // The color of the hint label and hint highlights

	Highlight1, Highlight2, Highlight3 string // The color of secondary highlights

	// Background and symbol colors for diffs
	DiffAddBackground, DiffAddForeground       string
	DiffDeleteBackground, DiffDeleteForeground string
}

var defaultColors = makeDefaultTokenColors()

func DefaultColorPalette() *ColorPalette {
	return &ColorPalette{
		TokenColors:  defaultColors,
		Type:         ansi.CodeCyan,
		Function:     ansi.CodeMagenta,
		StringEscape: ansi.CodeCyan,
		BuiltinFunc:  ansi.CodeBlue,

		Divider:  ansi.CodeDim,
		Box:      ansi.CodeBlue,
		FileName: ansi.CodeCyan,
		FilePos:  ansi.CodeYellow,

		ErrorColor:   ansi.CodeBrightRed,
		WarningColor: ansi.CodeBrightYellow,
		HintColor:    ansi.CodeBrightBlue,

		Highlight1: ansi.CodeBrightGreen,
		Highlight2: ansi.CodeBrightMagenta,
		Highlight3: ansi.CodeBrightBlue,

		DiffAddBackground:    "",
		DiffAddForeground:    ansi.CodeBrightGreen,
		DiffDeleteBackground: "", // TODO
		DiffDeleteForeground: ansi.CodeBrightRed,
	}
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

	HighlightMultilineTL           rune
	HighlightMultilineBL           rune
	HighlightMultilineOffset       rune
	HighlightMultilineSkip         rune
	HighlightMultilineSkipEllipsis rune

	ArrowT  rune
	ArrowBL rune
	ArrowH  rune
	ArrowV  rune

	ErrorDivider rune
}

// DefaultCharacterSet returns a character set that uses default
// characters, which may be Unicode.
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

		HighlightMultilineTL:           '╭',
		HighlightMultilineBL:           '╰',
		HighlightMultilineOffset:       '·',
		HighlightMultilineSkip:         '├',
		HighlightMultilineSkipEllipsis: '·',

		ArrowT:  '┬',
		ArrowBL: '╰',
		ArrowH:  '─',
		ArrowV:  '│',

		ErrorDivider: '-',
	}
}

// ASCIICharacterSet returns a character set that uses ASCII characters.
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

		HighlightMultilineTL:           'r',
		HighlightMultilineBL:           '\'',
		HighlightMultilineOffset:       '.',
		HighlightMultilineSkip:         '=',
		HighlightMultilineSkipEllipsis: '.',

		ArrowT:  ',',
		ArrowBL: '`',
		ArrowH:  '-',
		ArrowV:  '|',

		ErrorDivider: '-',
	}
}

// makeDefaultTokenColors returns a map of token types to their default colors.
func makeDefaultTokenColors() map[lexer.TokenType]string {
	const (
		colorKeyword  = ansi.CodeRed
		colorOperator = colorKeyword
		colorNumber   = ansi.CodeYellow
		colorString   = ansi.CodeGreen
		colorBoolean  = colorNumber
		colorComment  = ansi.CodeDim
		colorPunct    = ansi.CodeGray
		colorType     = ansi.CodeCyan
		colorIllegal  = ansi.CodeReset
	)
	colors := map[lexer.TokenType]string{
		lexer.Type:    colorKeyword,
		lexer.Func:    colorKeyword,
		lexer.String:  colorString,
		lexer.Regex:   colorString,
		lexer.Numeric: colorNumber,
		lexer.Boolean: colorBoolean,
		lexer.Nil:     colorBoolean,
		lexer.Illegal: colorIllegal,
		lexer.And:     colorOperator,
		lexer.Or:      colorOperator,
		// Comments
		lexer.BlockComment: colorComment,
		lexer.LineComment:  colorComment,
		lexer.Hashbang:     colorComment,
		// Punctuation
		lexer.Dot:                colorPunct,
		lexer.Colon:              colorPunct,
		lexer.Comma:              colorPunct,
		lexer.LeftCurlyBrace:     colorPunct,
		lexer.RightCurlyBrace:    colorPunct,
		lexer.LeftParenthesis:    colorPunct,
		lexer.RightParenthesis:   colorPunct,
		lexer.LeftBracket:        colorPunct,
		lexer.RightBracket:       colorPunct,
		lexer.At:                 colorPunct,
		lexer.HashLeftCurlyBrace: colorPunct,
		lexer.Hash:               colorPunct,
	}
	for _, op := range lexer.OperatorMap {
		if _, ok := colors[op]; !ok {
			colors[op] = colorOperator
		}
	}
	for _, kw := range lexer.KeywordMap {
		if _, ok := colors[kw]; !ok {
			colors[kw] = colorKeyword
		}
	}
	return colors
}
