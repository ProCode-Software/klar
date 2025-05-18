package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseStatement() ast.ASTItem {
	kind := p.CurrentTokenKind()
	result, handled := p.handleStatement(kind)
	if handled {
		return result
	}
	// Then it is an expression
	expr := p.ParseExpression(DefaultBindingPower)
	p.Expect(lexer.EndOfStatement)
	return ast.ExpressionStatement{Expression: expr}
}
