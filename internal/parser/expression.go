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
		// Tuple (requires at least one comma)
		tuple := ast.TupleLiteral{}
		tuple.Values = append(tuple.Values, expr)
		p.Advance()
		for p.IsNot(lexer.RightParenthesis) {
			tuple.Values = append(tuple.Values, p.ParseExpression(LogicalBindingPower))
			if p.CurrentTokenKind() != lexer.RightParenthesis {
				p.Expect(lexer.Comma)
			}
		}
		p.Expect(lexer.RightParenthesis)
		return tuple
	case lexer.RightParenthesis:
		// Grouped expression
		p.Advance()
		return expr
	default:
		panic(errors.ExpectedTokenError(lexer.RightParenthesis, next, next.Position))
	}
}

func (p *Parser) ParseMap() ast.MapLiteral {
	p.Expect(lexer.HashLeftCurlyBrace)
	entries := []ast.Pair{}
	for p.IsNot(lexer.RightCurlyBrace) {
		entry := ast.Pair{
			Key: p.ParseExpression(LogicalBindingPower),
		}
		p.Expect(lexer.Colon)
		entry.Value = p.ParseExpression(LogicalBindingPower)
		entries = append(entries, entry)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.MapLiteral{Entries: entries}
}
