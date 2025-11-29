package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) parseCaseSubExpr() ast.Expression {
	tok := p.Curr()
	var res ast.Expression
outer:
	switch tok.Kind {
	// Relational operators don't need explicit LHS
	// 	when x {
	// 		< 5 -> ...
	// }
	case lexer.EqualEqual, lexer.NotEqual, lexer.LessThan, lexer.GreaterThan,
		lexer.GreaterEqualTo, lexer.LessEqualTo, lexer.In, lexer.NotIn:
		res = p.ParseBinaryExpression(nil, bpOf(tok.Kind))
	case lexer.Question:
		p.Advance()
		res = &ast.NilLiteral{Shorthand: true}
	case lexer.Underscore:
		p.Advance()
		res = &ast.Discard{}
	case lexer.NotCan:
		res = p.ParseWhenCan()
	case lexer.Can:
		switch peek := p.PeekKind(); peek {
		default:
			if !isValidIdentifier(peek) {
				break
			}
			fallthrough
		case lexer.LeftParenthesis, lexer.LeftBracket, lexer.Identifier,
			lexer.Stroke, lexer.Ellipsis:
			res = p.ParseWhenCan()
			break outer
		}
		fallthrough
	default:
		res = p.ParseExpression(ExpressionBindingPower)
	}
	return markStartEndPos(p, res, tok.Position)
}

func (p *Parser) ParseWhenCan() *ast.WhenCanCase {
	op := newOperator(p.Advance())            // can, !can
	typ := p.ParseType(UnionTypeBindingPower) // Don't include '|'
	// Parse types with lower binding power than '|': '?' and '...'
	switch curr := p.CurrKind(); curr {
	case lexer.Question, lexer.Ellipsis:
		typ = p.ParseTypeLED(typ, TypeBindingPowerMap[curr])
	}
	when := &ast.WhenCanCase{Operator: op, Type: typ}
	if p.CurrKind() == lexer.LeftParenthesis {
		params := p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis))
		when.Params = params.Args
	}
	return when
}

func (p *Parser) parseWhenCase(subjects int) *ast.WhenCase {
	var (
		c         = &ast.WhenCase{}
		commaExp  = make([]ast.Expression, 0, subjects)
		orOpts    [][]ast.Expression
		braceLine uint32
	)
	// Back up isWhenCase flag
	oldIsWhenCase := p.flags & isWhenCase
	p.flags |= isWhenCase
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
	p.flags = (p.flags &^ isWhenCase) | oldIsWhenCase // Restore old isWhenCase flag
	p.Expect(lexer.Arrow)
	switch p.CurrKind() {
	case lexer.LeftCurlyBrace:
		c.Body = p.ParseBlock()
		c.Braces = true
		braceLine = c.Body.(*ast.Block).Range.End.Line

		if k := p.Curr(); k.Kind != lexer.RightCurlyBrace &&
			!isImplicitWhenOp(braceLine, k) {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	default:
		// BUG: Braces/comma required before '<' starting next case
		res := p.ParseStatement()
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
		if p.CurrKind() == lexer.Comma {
			p.Advance()
		}
	}
	return c
}

func isImplicitWhenOp(prevLine uint32, t lexer.Token) bool {
	switch t.Kind {
	case lexer.Comma, lexer.EndOfStatement:
		return false
	case lexer.EqualEqual, lexer.NotEqual, lexer.LessThan, lexer.GreaterThan,
		lexer.GreaterEqualTo, lexer.LessEqualTo, lexer.In, lexer.NotIn, lexer.Question:
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
