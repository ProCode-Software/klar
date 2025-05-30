package parser

import (
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type EscapeMap = map[lexer.Position]ast.StringEscape

func (p *Parser) parseStringEscapes(tok lexer.Token) EscapeMap {
	var (
		lexEscapes = tok.Attributes["escapes"].(lexer.EscapeMap)
		escapes    = make(EscapeMap, len(lexEscapes))
	)
	if len(lexEscapes) == 0 {
		return nil
	}
	for i, e := range lexEscapes {
		var (
			val      ast.StringEscape
			src      = e.Value
			parseHex = func(str string) int32 {
				return int32(unwrap(strconv.ParseInt(str, 16, 32)))
			}
		)
		if e.Invalid > 0 {
			errPos := ranges.AddPosition(tok.Position, e.ErrorPosition)
			p.Error(errors.InvalidEscapeError(e, errPos))
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
		escapes[i] = val
	}
	return escapes
}
