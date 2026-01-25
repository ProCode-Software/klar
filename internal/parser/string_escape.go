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
			p.Error(&ParseError{
				Range:     ranges.Offset(ranges.Sub(*e.ErrorPosition, 0, 1), 0, 1),
				ErrorCode: errors.ErrStringEscape,
				Params: errors.ErrorParams{
					"reason": e.Invalid,
					"type":   e.Type,
					"escape": e.Value,
				},
			})
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
				p.Error(errors.Range(errors.ErrUnicodeEscapeTooBig, ranges.Range{
					ranges.Add(e.Pos, 0, 3),
					ranges.Add(e.Pos, 0, uint32(len(src)-1)), //nolint:gosec
				}))
				frags[i] = ast.EscapeFragment{ast.BadEscape{Source: src}}
				continue
			}
			val.Value = ast.HexadecimalEscape{Hex: hex}
		case lexer.EscInterpolation:
			ct := p.parseStringInterpolation(*e.Interpolated)
			val.Value = ast.StringInterpolation{Expression: ct}
		}
		frags[i] = val
	}
	return frags
}

func (pBase *Parser) parseStringInterpolation(content []lexer.Token) (res ast.Node) {
	content = append(content, lexer.Token{
		Kind:     lexer.EOF,
		Position: ranges.TokenEnd(content[len(content)-1]),
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
