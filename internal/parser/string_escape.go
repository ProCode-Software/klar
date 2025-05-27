package parser

import (
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func parseStringEscapes(token lexer.Token) []ast.StringEscape {
	lexEscapes := token.Attributes["escapes"].(map[lexer.Position]lexer.StringEscape)
	escapes := make([]ast.StringEscape, 0, len(lexEscapes))
	for pos, escape := range lexEscapes {
		var escapeValue ast.StringEscapeValue
		src := escape.Value
		if escape.Invalid > 0 {
			panic(errors.InvalidEscapeError(escape.Invalid, pos, src))
		}
		switch escape.Type {
		case lexer.EscCharacter:
			escapeValue = ast.CharacterEscape{Character: rune(src[1])}
		case lexer.HexadecimalEscape:
			escapeValue = ast.HexadecimalEscape{
				Hex: int32(unwrap(strconv.ParseInt(src, 16, 32))),
			}
		case lexer.EscUnicode:
			escapeValue = ast.HexadecimalEscape{
				Hex: int32(unwrap(strconv.ParseInt(src[3:len(src)-4], 16, 32))),
			}
		case lexer.EscInterpolation:
			escapeValue = ast.StringInterpolation{} // TODO: lex string interpolation
		}
		escapes = append(escapes, ast.StringEscape{
			Index: pos,
			Type:  escape.Type,
			Value: escapeValue,
		})
	}
	return escapes
}
