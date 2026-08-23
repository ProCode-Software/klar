package reporter

import (
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/util"
)

// makeDefaultTokenColors returns a map of token types to their default colors.
func makeDefaultTokenColors() map[lexer.TokenType]string {
	const (
		colorKeyword  = ansi.CodeRed
		colorOperator = colorKeyword
		colorNumber   = ansi.CodeYellow
		colorString   = ansi.CodeGreen
		colorBoolean  = colorNumber
		colorComment  = ansi.CodeDim // TODO: Italic?
		colorPunct    = ansi.CodeGray
		colorIllegal  = ansi.CodeReset
	)
	gen := &TokenColorGenerator{
		Keyword:  colorKeyword,
		Operator: colorOperator,
		Number:   colorNumber,
		String:   colorString,
		Boolean:  colorBoolean,
		Comment:  colorComment,
		Punct:    colorPunct,
		Illegal:  colorIllegal,
	}
	return gen.Generate()
}

// TODO: other palettes: frost, github

// FrostColorPalette returns a color palette based on the [Frost] theme by ProCode Software.
//
// [Frost]: https://github.com/ProCode-Software/vscode-themes
func FrostColorPalette() *ColorPalette {
	// See https://github.com/ProCode-Software/vscode-themes/blob/main/frost-theme/src/frost-dark-color-theme.json
	var (
		white      = ansi.RGB(util.HexToRGB("#c7d1dd"))
		dim        = ansi.RGB(util.HexToRGB("#728195")) // --editor-fg
		boolean    = ansi.RGB(util.HexToRGB("#6897cc"))
		comment    = ansi.RGB(util.HexToRGB("#5b6776"))
		escape     = ansi.RGB(util.HexToRGB("#70a9ea"))
		function   = ansi.RGB(util.HexToRGB("#8cb6e2"))
		globalFunc = ansi.RGB(util.HexToRGB("#90a9c5")) // --t-global
		keyword    = ansi.RGB(util.HexToRGB("#ffffff"))
		stringLit  = ansi.RGB(util.HexToRGB("#a3c8f3"))
		typeName   = globalFunc
	)
	// UI colors
	var (
		red     = ansi.RGB(util.HexToRGB("#fe838f"))
		green   = ansi.RGB(util.HexToRGB("#d0ff92"))
		yellow  = ansi.RGB(util.HexToRGB("#fed583"))
		orange  = ansi.RGB(util.HexToRGB("#fbb07a"))
		magenta = ansi.RGB(util.HexToRGB("#ffa4f3")) // --ansi-magenta
		accent  = ansi.RGB(util.HexToRGB("#99c4f8"))
		blue    = ansi.RGB(util.HexToRGB("#6bceff")) // --ansi-blue
		cyan    = ansi.RGB(util.HexToRGB("#78f2e6")) // --ansi-cyan
		// namespace = ansi.RGB(util.HexToRGB("#A1BEDC"))
	)
	return &ColorPalette{
		TokenColors: (&TokenColorGenerator{
			Ident:    white,
			Keyword:  keyword,
			Operator: keyword,
			Number:   boolean,
			String:   stringLit,
			Boolean:  boolean,
			Comment:  comment + ansi.CodeItalic,
			Punct:    dim,
			Illegal:  red + ansi.CodeItalic,
		}).Generate(),
		Type:         typeName,
		Function:     function,
		BuiltinFunc:  globalFunc,
		StringEscape: escape,

		ErrorColor:   red + ansi.CodeBold,
		WarningColor: yellow + ansi.CodeBold,
		HintColor:    ansi.CodeBold + blue,

		FileName:   cyan,
		FilePos:    yellow,
		Divider:    dim,
		Box:        accent,
		Highlight1: magenta,
		Highlight2: orange,
		Highlight3: green,

		DetailBox:   magenta,
		DetailColor: ansi.CodeBold + magenta,

		DiffAdd:              green,
		DiffDelete:           red,
		DiffAddBackground:    green + ansi.CodeBold,
		DiffDeleteBackground: red + ansi.CodeBold,
	}
}
