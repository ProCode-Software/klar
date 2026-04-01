package parser

import (
	"strconv"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func parseHex(str string) int32 {
	//nolint:gosec // There is a length limit in the lexer
	return int32(tryStrconv(strconv.ParseInt(str, 16, 32)))
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
			p.Error(&ParseError{
				Range:     ranges.Offset(err.Pos.Sub(0, 1), 0, 1),
				ErrorCode: errors.ErrStringEscape,
				Params: errors.ErrorParams{
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
				p.Error(errors.Range(errors.ErrUnicodeEscapeTooBig, ranges.Range{
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

func (pBase *Parser) parseStringInterpolation(content []lexer.Token) (res ast.Node) {
	content = append(content, lexer.Token{
		Kind:     lexer.EOF,
		Position: content[len(content)-1].End(),
	})
	p := New(content, &pBase.Options)
	defer p.Reset()
	p.InsertEOS()
	// Allow type pattern matching in when cases
	// when str {
	//	"Hello {_}" -> ...
	//	"{x: Int} cats" -> ...
	// }
	if pBase.isWhenCase() && p.PeekKind() == lexer.Colon {
		name := p.ParseIdentOrDiscard()
		p.Expect(lexer.Colon)
		typ := p.ParseType(DefaultTypeBindingPower)
		res = &ast.StringTypeMatch{
			BaseNode: newBaseNode(name.Position, p.lastTokEnd()),
			Name:     name,
			Type:     typ,
		}
	} else {
		res = p.ParseExpression(ExpressionBindingPower)
	}
	// Copy errors back
	pBase.Errors = append(pBase.Errors, p.Errors...)
	// Check that there is nothing else
	if c := p.CurrKind(); c != lexer.Newline && c != lexer.EOF {
		pBase.Error(errors.Token(errors.ErrExpectedInterpolationEnd, p.Curr()))
	}
	return res
}
