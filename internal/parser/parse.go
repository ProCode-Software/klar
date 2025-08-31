package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Parse parses p.Tokens into a [*ast.Program].
func (p *Parser) Parse() *ast.Program {
	defer func() {
		switch err := recover(); err {
		case nil, stopParsing{}:
			return
		default:
			panic(err)
		}
	}()
	body := make([]ast.Statement, 0, len(p.Tokens)/2)
	comments := p.RemoveComments() // Move comments
	p.InsertEOS()                  // Add the "semicolons"
	for p.HasTokens() {
		if p.Options.StopOnError && len(p.Errors) > 0 {
			break
		}
		if p.CurrKind() == lexer.EndOfStatement {
			p.Advance()
		}
		body = append(body, p.ParseTopLevelStatement())
	}
	prog := &ast.Program{Body: body, Comments: comments}
	prog.SetPos(p.Tokens[0].Position, p.Tokens[len(p.Tokens)-1].Position)
	return prog
}

func (p *Parser) unknownTokenErr() {
	p.Error(errors.UnexpectedToken(p.AdvanceNonBoundary()))
	p.skipUntilBoundary()
}

func (p *Parser) ParseExpression(bp BindingPower) ast.Expression {
	expr := p.ParseFull(bp)
	if expr, ok := expr.(ast.Expression); ok {
		return expr
	}
	p.errExpectedExpr(expr)
	return &ast.BadExpression{Value: expr}
}

func (p *Parser) ParseFull(bp BindingPower) ast.Node {
	kind := p.CurrKind()
	if kind == lexer.EOF {
		return &ast.BadExpression{Token: kind}
	}
	left, handled := p.handleNUD(kind)
	if !handled {
		p.unknownTokenErr()
		return &ast.BadExpression{Token: kind}
	}
	return p.ParseLED(left, bp)
}

func (p *Parser) ParseLED(left ast.Node, bp BindingPower) ast.Node {
	var handled bool
	for BindingPowerMap[p.CurrKind()] > bp {
		kind := p.CurrKind()
		left, handled = p.handleLED(kind, left, BindingPowerMap[kind])
		if !handled {
			p.unknownTokenErr()
			return &ast.BadExpression{
				Token: kind,
				Value: left,
			}
		}
	}
	// left = left.SetPos(left.GetRange().Start, p.savePos())
	return left
}

func (p *Parser) ParseTopLevelStatement() ast.Statement {
	kind := p.CurrKind()
	result, handled := p.handleStatement(kind, true)
	if handled {
		if kind != lexer.Public {
			p.Expect(lexer.EndOfStatement)
		}
		return result
	}
	return p.ParseStatement()
}

func (p *Parser) ParseStatement() ast.Statement {
	kind := p.CurrKind()
	var res ast.Node
	res, handled := p.handleStatement(kind, false)
	if handled {
		p.Expect(lexer.EndOfStatement)
		return res.(ast.Statement)
	}
	res, handled = p.handleStatementNUD(kind)
	if handled {
		res = p.ParseLED(res, DefaultBindingPower)
	} else {
		res = p.ParseFull(DefaultBindingPower)
	}
	p.Expect(lexer.EndOfStatement)
	switch res := res.(type) {
	// Left-denoted statement
	case ast.Statement:
		return res
	// Then it is an expression
	case ast.Expression:
		stmt := &ast.ExpressionStatement{Expression: res}
		copyPos(res, stmt)
		return stmt
	// I don't know what this is. If this occurs, then it is a bug.
	default:
		panic(fmt.Sprintf("node %v is neither an expression nor statement", res))
	}
}

func (p *Parser) skipUntilBoundary() {
	brackCount := 1
	for p.HasTokens() && p.CurrKind() != lexer.EndOfStatement {
		if p.CurrKind() == lexer.Comma && brackCount <= 1 {
			return
		}
		switch p.Advance().Kind {
		case lexer.LeftParenthesis, lexer.LeftBracket, lexer.LeftCurlyBrace, lexer.HashLeftCurlyBrace:
			brackCount++
		case lexer.RightParenthesis, lexer.RightBracket, lexer.RightCurlyBrace:
			brackCount--
			if brackCount <= 0 {
				return
			}
		}
	}
}

func parseSeriesWithBP[T ast.Node](
	p *Parser, arr *[]T,
	bp BindingPower, until, sepBy lexer.TokenType,
) {
	parseSeries(
		p, arr,
		func() T { return p.ParseExpression(bp).(T) },
		until, sepBy, false,
	)
}

func parseSeries[T ast.Node](
	p *Parser, arr *[]T,
	with func() T, until, separator lexer.TokenType,
	allowEOS bool,
) {
	if until == 0 && separator == 0 {
		panic("until and separator cannot both be zero")
	}
	for p.HasTokens() && p.CurrKind() != until {
		if !allowEOS && p.CurrKind() == lexer.EndOfStatement {
			break
		}
		start := p.Curr().Position
		*arr = append(*arr, markStartEndPos(p, with(), start))
		if !p.HasTokens() {
			break
		}
		curr := p.CurrKind()
		if curr == until || (until == 0 && curr != separator) ||
			(!allowEOS && curr == lexer.EndOfStatement) {
			break
		}
		if separator != 0 {
			p.Expect(separator)
		}
	}
	if until != 0 {
		p.Expect(until)
	}
}
