package parser

import (
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func parseHex(str string) int32 {
	val, err := strconv.ParseInt(str, 16, 32)
	if err != nil {
		// There is a length limit in the lexer, so there should be no overflow
		panic("strconv.ParseInt failed while parsing hex escape: " + err.Error())
	}
	//nolint:gosec
	return int32(val)
}

func (p *Parser) parseStringEscapes(tok lexer.Token) []ast.StringFragment {
	lexEscapes := tok.Attributes["params"].(lexer.StringAttrs).Fragments
	if len(lexEscapes) == 0 {
		return nil
	}
	frags := make([]ast.StringFragment, len(lexEscapes))
	for i, frag := range lexEscapes {
		// Text fragment
		if frag, ok := frag.(lexer.TextFragment); ok {
			frags[i] = frag
			continue
		}
		// Escape or interpolation fragment
		e := frag.(lexer.StringEscape)
		src := e.Value

		// Escape error
		if err := e.Error; err != nil {
			p.Error(&Error{
				Range: ranges.Offset(err.Pos.Sub(0, 1), 0, 1),
				Code:  klarerrs.ErrStringEscape,
				Params: klarerrs.ErrorParams{
					"reason": err.Code,
					"type":   e.Type,
					"escape": src,
				},
			})
			frags[i] = ast.EscapeFragment{ast.BadEscape{Source: src}}
			continue
		}

		frag := ast.EscapeFragment{}
		switch e.Type {
		// Interpolation
		case lexer.EscInterpolation:
			frags[i] = ast.InterpolationFragment{
				Expression: p.parseStringInterpolation(*e.Interpolated),
			}
			continue

		// Actual escapes
		case lexer.EscCharacter:
			frag.Value = ast.CharacterEscape{Character: rune(src[1])}
		case lexer.EscHex:
			frag.Value = ast.HexadecimalEscape{Hex: parseHex(src[2:])}
		case lexer.EscUnicode:
			hex := parseHex(src[3 : len(src)-1])
			if hex > 0x10FFFF {
				p.Error(klarerrs.Range(klarerrs.ErrUnicodeEscapeTooBig, ranges.Range{
					e.Pos.Add(0, 3),
					e.Pos.Add(0, uint32(len(src)-1)), //nolint:gosec
				}))
				frag.Value = ast.BadEscape{Source: src}
			} else {
				frag.Value = ast.HexadecimalEscape{Hex: hex}
			}
		}
		frags[i] = frag
	}
	return frags
}

func (p *Parser) parseStringInterpolation(content []lexer.Token) (res ast.Expression) {
	content = append(content, lexer.Token{
		Kind:     lexer.EOF,
		Position: content[len(content)-1].End(),
	})
	ip := New(content, &p.Options)
	defer ip.Reset()
	ip.InsertEOS()
	// Allow type pattern matching in when cases
	// when str {
	//	"Hello {_}" -> ...
	//	"{x: Int} cats" -> ...
	// }
	if p.isWhenCase() && ip.PeekKind() == lexer.Colon {
		name := ip.ParseIdentOrDiscard()
		ip.Expect(lexer.Colon)
		typ := ip.ParseType(DefaultTypeBindingPower)
		res = &ast.StringTypeMatch{
			BaseNode: newBaseNode(name.Position, ip.lastTokEnd()),
			Name:     name,
			Type:     typ,
		}
	} else {
		res = ip.ParseExpression(ExpressionBindingPower)
	}
	// Copy errors back
	p.Errors = append(p.Errors, ip.Errors...)
	// Check that there is nothing else
	if c := ip.CurrKind(); c != lexer.Newline && c != lexer.EOF {
		p.Error(klarerrs.Token(klarerrs.ErrExpectedInterpolationEnd, ip.Curr()))
	}
	return res
}
