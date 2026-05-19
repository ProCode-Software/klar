package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) parseCaseSubExpr() ast.Expression {
	tok := p.Curr()
	var res ast.Expression
	switch tok.Kind {
	// Relational operators don't need explicit LHS
	// 	when x {
	// 		< 5 -> ...
	// }
	case lexer.EqualEqual, lexer.NotEqual, lexer.LessThan, lexer.GreaterThan,
		lexer.GreaterEqualTo, lexer.LessEqualTo, lexer.In, lexer.NotIn:
		res = p.ParseBinaryExpression(nil, bpOf(tok.Kind))
	case lexer.Underscore:
		p.Advance()
		res = &ast.Discard{}
	default:
		res = p.ParseExpression(ExpressionBindingPower)
	}
	return markStartEndPos(p, res, tok.Position)
}

func (p *Parser) parseWhenCase(subjects int) *ast.WhenCase {
	var (
		c        = &ast.WhenCase{}
		commaExp = make([]ast.Expression, 0, subjects)
		orOpts   [][]ast.Expression
	)
	// Back up isWhenCase flag
	oldIsWhenCase := p.flags & isWhenCase
	p.flags |= isWhenCase
	defer func() { p.flags = oldIsWhenCase }()
	// ',' binds tighter than '|' in case
loop:
	for p.HasTokens() {
		if p.CurrKind() == lexer.Stroke {
			p.Advance()
		}
		commaExp = append(commaExp, p.parseCaseSubExpr())
		switch p.CurrKind() {
		case lexer.Stroke:
			orOpts = append(orOpts, commaExp)
			clear(commaExp)
			commaExp = commaExp[:0]
			p.Advance()
		case lexer.If, lexer.Arrow:
			orOpts = append(orOpts, commaExp)
			break loop
		case lexer.Comma:
			p.Advance()
		default:
			break loop
		}
	}
	c.Options = orOpts
	// Guard clause
	// 	when x, y {
	//		5, _ if y < 10 -> ...
	// 	}
	if p.CurrKind() == lexer.If {
		p.Advance()
		c.Guard = p.ParseExpression(ExpressionBindingPower)
	}
	p.flags = oldIsWhenCase
	p.Expect(lexer.Arrow)
	switch p.CurrKind() {
	// Block
	case lexer.LeftCurlyBrace:
		c.Body = p.ParseBlock()
		c.Braces = true
		braceLine := c.Body.(*ast.Block).Range.End.Line

		if k := p.Curr(); k.Kind != lexer.RightCurlyBrace &&
			!isImplicitWhenOp(braceLine, k) {
			p.ExpectOneOf(lexer.Newline, lexer.Comma)
		}
	// Statement/expression outside braces
	default:
		switch p.CurrKind() {
		// Treat these tokens as expressions
		case lexer.For, lexer.Func:
			c.Body = p.ParseExpression(ExpressionBindingPower)
			p.ExpectOneOf(lexer.Newline, lexer.Comma)
			return c
		}
		// BUG: Braces/comma required before '<' starting next case
		res := p.ParseStatement(allowCommaTerminator)
		switch res := res.(type) {
		// All expressions are allowed
		case *ast.ExpressionStatement:
			c.Body = res.Expression
		// Allow some kinds of statements outside of braces
		case *ast.AssignmentStatement, *ast.ReturnStatement,
			*ast.NextStatement, *ast.UpdateStatement, *ast.StopStatement:
			c.Body = res
		default:
			// Expected expression error
			p.Error(errors.Node(errors.ErrRequiredBraces, res))
			c.Body = &ast.BadExpression{Value: res}
		}
	}
	return c
}

func isImplicitWhenOp(prevLine uint32, t lexer.Token) bool {
	switch t.Kind {
	case lexer.Comma, lexer.Newline:
		return false
	case lexer.EqualEqual, lexer.NotEqual, lexer.LessThan, lexer.GreaterThan,
		lexer.GreaterEqualTo, lexer.LessEqualTo, lexer.In, lexer.NotIn,
		lexer.Question, lexer.Dot:
		return t.Position.Line != prevLine
	}
	return false
}

func (p *Parser) ParseWhenBlock() *ast.WhenExpression {
	p.Expect(lexer.When)
	w := &ast.WhenExpression{}
	if p.CurrKind() != lexer.LeftCurlyBrace {
		// Subjects
		parseExprSeries(
			p, &w.Subjects, ExpressionBindingPower,
			lexer.LeftCurlyBrace, lexer.Comma,
		)
	} else {
		p.Expect(lexer.LeftCurlyBrace)
	}
	lenSubj := len(w.Subjects)
	parseSeries(
		p, &w.Cases,
		func() *ast.WhenCase { return p.parseWhenCase(lenSubj) },
		lexer.RightCurlyBrace, 0, true,
	)
	return w
}
