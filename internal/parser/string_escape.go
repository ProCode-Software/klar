package parser

import (
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func parseStringEscapes(tok lexer.Token) []ast.StringEscape {
	var (
		lexEscapes = tok.Attributes["escapes"].(map[int]lexer.StringEscape)
		escapes    = make([]ast.StringEscape, 0, len(lexEscapes))
	)
	for i, e := range lexEscapes {
		var (
			val      ast.StringEscapeValue
			src      = e.Value
			parseHex = func(str string) int32 {
				return int32(unwrap(strconv.ParseInt(str, 16, 32)))
			}
		)
		if e.Invalid > 0 {
			errPos := ranges.Add(tok.Position, 0, e.ErrorPosition)
			panic(errors.InvalidEscapeError(e, errPos))
		}
		switch e.Type {
		case lexer.EscCharacter:
			val = ast.CharacterEscape{Character: rune(src[1])}
		case lexer.EscHex:
			val = ast.HexadecimalEscape{Hex: parseHex(src[2:])}
		case lexer.EscUnicode:
			val = ast.HexadecimalEscape{Hex: parseHex(src[3 : len(src)-1])}
		case lexer.EscInterpolation:
			val = ast.StringInterpolation{} // TODO: lex string interpolation
		}
		escapes = append(escapes, ast.StringEscape{
			Index: i,
			Type:  e.Type,
			Value: val,
		})
	}
	return escapes
}
