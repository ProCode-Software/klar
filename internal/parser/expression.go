package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseBinaryExpression(left ast.ASTItem, bp BindingPower) ast.BinaryExpression {
	op := p.Advance()
	right := p.ParseExpression(bp)
	return ast.BinaryExpression{
		Left:     left,
		Operator: op.Kind,
		Right:    right,
	}
}

func (p *Parser) ParseUnaryExpression() ast.UnaryExpression {
	op := p.Advance().Kind
	right := p.ParseExpression(UnaryBindingPower)
	return ast.UnaryExpression{
		Operator: op,
		Right:    right,
	}
}

func (p *Parser) ParseGroupOrTuple() ast.Expression {
	p.Advance() // (
	expr := p.ParseExpression(DefaultBindingPower)
	next := p.CurrentToken()
	switch next.Kind {
	case lexer.Comma:
		// Tuple
		panic("TODO")
	case lexer.RightParenthesis:
		// Grouped expression
		p.Advance()
		return expr
	default:
		panic(errors.ExpectedTokenError(lexer.RightParenthesis, next, next.Position))
	}
}

func (p *Parser) ParseTuple() ast.Expression {
	p.Advance() // (
	expr := p.ParseExpression(DefaultBindingPower)
	p.Expect(lexer.RightParenthesis)
	return expr
}