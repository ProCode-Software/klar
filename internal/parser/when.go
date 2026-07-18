package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
)

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
		func() *ast.WhenCase { return p.parseWhenCase(max(lenSubj, 1)) },
		lexer.RightCurlyBrace, 0, true,
	)
	return w
}

func (p *Parser) parseWhenCase(subjects int) *ast.WhenCase {
	c := &ast.WhenCase{}
	subjectPatterns := make([]ast.Expression, 0, subjects)
	insertOption := func() {
		if len(subjectPatterns) != subjects {
			p.Error(
				klarerrs.Slice(klarerrs.ErrWrongSubjectCount, subjectPatterns).
					SetParam("expected", subjects).
					SetParam("got", len(subjectPatterns)),
			)
		}
		c.Options = append(c.Options, subjectPatterns)
		subjectPatterns = subjectPatterns[:0]
	}
	// Patterns and options
	func() {
		defer func(old uint8) { p.flags = old }(p.flags)
		p.flags |= whenPattern
		if p.CurrKind() == lexer.Stroke {
			p.Advance()
		}
		// ',' binds tighter than '|' in case
	loop:
		for p.HasTokens() {
			subjectPatterns = append(subjectPatterns, p.parseCaseSubExpr())
			switch p.CurrKind() {
			case lexer.Stroke:
				insertOption()
				p.Advance()
			case lexer.If, lexer.Arrow, lexer.As:
				insertOption()
				break loop
			case lexer.Comma:
				p.Advance()
			default:
				break loop
			}
		}
	}()

	// when a, b, c {
	//   < 5, < 3, < 0 as x, y, z | > 10, > 20, > 30 as x, y, z -> ...
	// }
	if p.CurrKind() == lexer.As {
		p.Advance()
		parseSeries(p, &c.As, p.ParseIdentOrDiscard, 0, lexer.Comma, false)
		if len(c.As) > subjects {
			p.ErrorLabelled(
				klarerrs.Slice(klarerrs.ErrWrongSubjectCount, c.As[subjects:]).
					SetParam("expected", subjects).SetParam("got", len(c.As)),
				"Extra subjects",
			)
		}
	}

	// Guard clause
	// 	when x, y {
	//		5, _ if y < 10 -> ...
	// 	}
	if p.CurrKind() == lexer.If {
		p.Advance()
		c.Guard = p.ParseExpression(ExpressionBindingPower)
	}

	// Body
	p.Expect(lexer.Arrow)
	switch p.CurrKind() {
	// Block
	case lexer.LeftCurlyBrace:
		c.Body = p.ParseBlock()
		braceLine := c.Body.(*ast.Block).Range.End.Line

		if k := p.Curr(); k.Kind != lexer.RightCurlyBrace &&
			!isImplicitWhenOp(braceLine, k) {
			p.ExpectOneOf(lexer.Newline, lexer.Comma)
		}
	// Statement/expression outside braces
	case lexer.For, lexer.Func:
		// Treat these tokens as expressions
		c.Body = p.ParseExpression(ExpressionBindingPower)
		p.ExpectOneOf(lexer.Newline, lexer.Comma)
		return c
	default:
		// BUG: Braces/comma required before '<' starting next case
		stmt := p.ParseStatement(allowCommaTerminator)
		switch stmt := stmt.(type) {
		// All expressions are allowed
		case *ast.ExpressionStatement:
			c.Body = stmt.Expression
		// Allow some kinds of statements outside of braces
		case *ast.AssignmentStatement, *ast.ReturnStatement,
			*ast.NextStatement, *ast.StopStatement:
			c.Body = stmt
		default:
			// Expected expression error
			p.Error(klarerrs.Node(klarerrs.ErrRequiredBraces, stmt))
			c.Body = &ast.BadExpression{Value: stmt}
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
		lexer.Dot, lexer.Ellipsis, lexer.DotDotLessThan:
		return t.Position.Line != prevLine
	}
	return false
}

func (p *Parser) parseCaseSubExpr() ast.Expression {
	tok := p.Curr()
	var res ast.Expression
	switch tok.Kind {
	// Relational operators don't need explicit LHS
	// 	when x {
	// 		< 5 -> ...
	//  }
	case lexer.EqualEqual, lexer.NotEqual, lexer.In, lexer.NotIn:
		res = p.ParseBinaryExpression(nil, bpOf(tok.Kind))
	case lexer.LessThan, lexer.LessEqualTo, lexer.GreaterEqualTo, lexer.GreaterThan:
		res = p.ParseRelationalExpression(nil, bpOf(tok.Kind))
	case lexer.Underscore:
		p.Advance()
		res = &ast.Discard{}
	default:
		res = p.ParseExpressionFilter(func(tt lexer.TokenType) bool {
			return tt == lexer.Stroke || tt == lexer.As
		}, bpOf(lexer.Stroke), 0)
	}
	return markStartEndPos(p, res, tok.Position)
}

func (p *Parser) ParseAs(left ast.Expression) *ast.AsExpression {
	p.Advance()
	return &ast.AsExpression{Expression: left, Name: p.ParseIdentifier()}
}

func (p *Parser) ParseSubOptions(first ast.Expression) *ast.SubOptions {
	opts := &ast.SubOptions{Options: []ast.Expression{first}}
	p.Advance() // |
	parseExprSeries(p, &opts.Options, ExpressionBindingPower, 0, lexer.Stroke)
	return opts
}
