package parser

import (
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func parseHex(str string) int32 {
	return int32(unwrap(strconv.ParseInt(str, 16, 32)))
}

func (p *Parser) parseStringEscapes(tok lexer.Token) []ast.StringFragment {
	var (
		lexEscapes = tok.Attributes["params"].(lexer.StringAttrs).Fragments
		frags      = make([]ast.StringFragment, len(lexEscapes))
		val        = ast.EscapeFragment{}
	)
	if len(lexEscapes) == 0 {
		return nil
	}
	for i, frag := range lexEscapes {
		if frag, ok := frag.(lexer.TextFragment); ok {
			frags[i] = frag
			continue
		}
		e := frag.(lexer.StringEscape)
		src := e.Value
		if e.Invalid > 0 {
			p.Error(errors.StringEscape(e))
			frags[i] = ast.EscapeFragment{ast.BadEscape{Source: src}}
			continue
		}
		switch e.Type {
		case lexer.EscCharacter:
			val.Value = ast.CharacterEscape{Character: rune(src[1])}
		case lexer.EscHex:
			val.Value = ast.HexadecimalEscape{Hex: parseHex(src[2:])}
		case lexer.EscUnicode:
			hex := parseHex(src[3 : len(src)-1])
			if hex > 0x10FFFF {
				p.Error(errors.Range(errors.ErrUnicodeEscTooBig, ranges.Range{
					ranges.Add(e.Pos, 0, 3),
					ranges.Add(e.Pos, 0, uint32(len(src)-1)),
				}))
				frags[i] = ast.EscapeFragment{ast.BadEscape{Source: src}}
				continue
			}
			val.Value = ast.HexadecimalEscape{Hex: hex}
		case lexer.EscInterpolation:
			p2, bp := p.newInterpParser(*e.Interpolated)
			val.Value = ast.StringInterpolation{
				Expression: p2.ParseExpression(bp),
			}
		}
		frags[i] = val
	}
	return frags
}

func (p *Parser) newInterpParser(tokens []lexer.Token) (*Parser, BindingPower) {
	// Add the EOF
	tokens = append(tokens, lexer.Token{
		Kind: lexer.EOF,
		// Position: tokens[len(tokens)-1].Attributes["end"].(lexer.Position),
	})
	ep := New(tokens, &p.Options)
	ep.InsertEOS()
	// Allow type pattern matching in when cases
	// when str {
	// 	 "{x: Int} cats" -> ...
	// }
	if p.isWhenCase {
		return ep, DefaultBindingPower
	}
	return ep, ExpressionBindingPower
}
