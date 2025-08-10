package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
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
