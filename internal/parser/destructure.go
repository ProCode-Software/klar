package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseDestructure() ast.Destructure {
	switch p.CurrentTokenKind() {
	case lexer.Identifier:
		
		rangeFromToken()
	case lexer.HashLeftCurlyBrace:
	case lexer.LeftBracket:
	case lexer.LeftParenthesis:

	}
	return nil
}

func (p *Parser) ParseDestructureSeries() []ast.Destructure {
	return nil
}