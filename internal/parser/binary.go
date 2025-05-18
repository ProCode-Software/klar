package parser

import "github.com/ProCode-Software/klar/internal/ast"

func (p *Parser) ParseBinaryExpression(left ast.ASTItem, bp BindingPower) ast.ASTItem {
	op := p.Advance()
	right := p.ParseExpression(bp)
	return ast.BinaryExpression{
		Left: left,
		Operator: op.Kind,
		Right: right,
	}
}