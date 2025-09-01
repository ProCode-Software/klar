package printer

import (
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Token colors
const (
	colorDefault  = ansi.CodeReset
	colorKeyword  = ansi.CodeRed
	colorOperator = colorKeyword
	colorNumber   = ansi.CodeYellow
	colorString   = ansi.CodeGreen
	escapeColor   = ansi.CodeCyan
	colorBoolean  = colorNumber
	colorComment  = ansi.CodeDim
	colorPunct    = ansi.CodeGray
	colorType     = ansi.CodeCyan
	colorFunc     = ansi.CodeMagenta
	colorBuiltin  = ansi.CodeBlue
	colorIllegal  = colorDefault
)

// Builtins
var builtinFuncs = map[string]struct{}{
	"print": {}, "panic": {}, "assert": {}, "TODO": {}, "unwrap": {},
}

// Default color theme
var defaultColors = map[lexer.TokenType]string{
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

// Create default colors
func init() {
	for _, op := range lexer.OperatorMap {
		if _, ok := defaultColors[op]; !ok {
			defaultColors[op] = colorOperator
		}
	}
	for _, kw := range lexer.KeywordMap {
		if _, ok := defaultColors[kw]; !ok {
			defaultColors[kw] = colorKeyword
		}
	}
}
