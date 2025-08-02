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
			p.Error(errors.StringEscape(e))
			escapes[i] = ast.BadEscape{Source: src}
			continue
		}
		switch e.Type {
		case lexer.EscCharacter:
			val = ast.CharacterEscape{Character: rune(src[1])}
		case lexer.EscHex:
			val = ast.HexadecimalEscape{Hex: parseHex(src[2:])}
		case lexer.EscUnicode:
			hex := parseHex(src[3 : len(src)-1])
			if hex > 0x10FFFF {
				p.Error(errors.Range(errors.ErrUnicodeEscTooBig, ranges.Range{
					ranges.Add(i, 0, 3),
					ranges.Add(i, 0, uint32(len(src)-1)),
				}))
				escapes[i] = ast.BadEscape{Source: src}
				continue
			}
			val = ast.HexadecimalEscape{Hex: hex}
		case lexer.EscInterpolation:
			p2, bp := p.newInterpParser(e.Interpolated)
			val = ast.StringInterpolation{
				Expression: p2.ParseExpression(bp),
			}
		}
		escapes[i] = val
	}
	return escapes
}

func (p *Parser) newInterpParser(tokens []lexer.Token) (*Parser, BindingPower) {
	// Add the EOF
	tokens = append(tokens, lexer.Token{
		Kind: lexer.EOF,
		// Position: tokens[len(tokens)-1].Attributes["end"].(lexer.Position),
	})
	ep := New(tokens, &p.Options)
	ep.RemoveComments()
	ep.InsertEOS()
	// Allow type pattern matching in when cases
	// when str {
	// 	 "{x: Int} cats" -> ...
	// }
	if p.isWhenCase {
		return ep, AssignBindingPower
	}
	return ep, ExpressionBindingPower
}
