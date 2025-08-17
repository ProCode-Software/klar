package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) isMapIdentifier() bool {
	return p.IsCurrently(ast.ReservedIdent...) ||
		p.IsCurrently(lexer.Identifier, lexer.Numeric, lexer.Boolean, lexer.Nil)
}

func (p *Parser) expectMapIdent() lexer.Token {
	if !p.isMapIdentifier() {
		return p.Expect(lexer.Identifier)
	}
	return p.Advance()
}

func (p *Parser) expectNonNumericMapIdent() lexer.Token {
	if !p.isMapIdentifier() || p.CurrentTokenKind() == lexer.Numeric {
		return p.Expect(lexer.Identifier)
	}
	return p.Advance()
}

func (p *Parser) ParseIdentifier() *ast.Symbol {
	tok := p.AdvanceNonBoundary()
	if tok.Kind != lexer.Identifier && !slices.Contains(ast.Modifiers, tok.Kind) {
		if slices.Contains(ast.ReservedIdent, tok.Kind) {
			p.Error(errors.Token(errors.ErrReservedKeyword, tok))
		} else {
			p.Error(errors.ExpectedToken(lexer.Identifier, tok))
		}
	}
	return rangeFromToken(&ast.Symbol{Identifier: tok.Source}, tok)
}

func (p *Parser) ParseMapIdentifier(includingNumber bool) *ast.Symbol {
	tok := p.AdvanceNonBoundary()
	kind := tok.Kind
	switch {
	case kind == lexer.Identifier:
		break
	case kind == lexer.Numeric && !includingNumber:
		fallthrough
	case !slices.Contains(ast.Modifiers, tok.Kind) &&
		!slices.Contains(ast.ReservedIdent, tok.Kind):
		p.Error(errors.ExpectedToken(lexer.Identifier, tok))
	}
	return rangeFromToken(&ast.Symbol{Identifier: tok.Source}, tok)
}
