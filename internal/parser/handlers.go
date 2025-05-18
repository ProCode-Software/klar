package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) handleNUD(kind lexer.TokenType) (ast.ASTItem, bool) {
	switch kind {
	case lexer.Identifier, lexer.String, lexer.Numeric, lexer.Boolean, lexer.Nil:
		return p.ParsePrimaryExpression(), true
	default:
		return nil, false
	}
}

func (p *Parser) handleLED(
	kind lexer.TokenType, left ast.ASTItem, bp BindingPower,
) (ast.ASTItem, bool) {
	switch kind {
	case lexer.Plus, lexer.Minus, lexer.Times, lexer.Divide, lexer.Modulo, lexer.Exponent:
		return p.ParseBinaryExpression(left, BindingPowerMap[p.CurrentTokenKind()]), true
	default:
		return nil, false
	}
}

func (p *Parser) handleStatement(kind lexer.TokenType) (ast.Statement, bool) {
	return nil, false // TODO: add statements
}