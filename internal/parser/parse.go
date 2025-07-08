package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Parse parses tokens into a Program. If continueOnErr is true, the parser will
// not stop parsing on a syntax error.
func (p *Parser) Parse() *ast.Program {
	var (
		body     = make([]ast.Statement, 0, len(p.Tokens)/2)
		comments = p.RemoveComments() // Move comments
	)
	p.InsertEOS() // Add the "semicolons"
	for p.HasTokens() {
		if !p.Options.ContinueOnError && len(p.Errors) > 0 {
			break
		}
		if p.CurrentTokenKind() == lexer.EndOfStatement {
			p.Index++
			continue
		}
		body = append(body, p.ParseTopLevelStatement())
	}
	prog := &ast.Program{Body: body, Comments: comments}
	prog.SetPos(p.Tokens[0].Position, p.Tokens[len(p.Tokens)-1].Position)
	return prog
}

func (p *Parser) unknownTokenErr() {
	p.Error(errors.UnexpectedToken(p.CurrentToken()))
	if p.CurrentTokenKind() != lexer.EOF {
		p.Advance()
	}
}

func (p *Parser) errExpectedExpr(got ast.Node) {
	p.Error(errors.ParseError{
		ErrorCode: errors.ErrExpectedExpression,
		Node:      got,
		Range:     got.GetRange(),
	})
}

func (p *Parser) ParseExpression(bp BindingPower) ast.Expression {
	expr := p.ParseLED(bp)
	if _, ok := expr.(ast.Expression); !ok {
		p.errExpectedExpr(expr)
		return &ast.BadExpression{Value: expr}
	}
	return expr.(ast.Expression)
}

func (p *Parser) ParseLED(bp BindingPower) ast.Node {
	kind := p.CurrentTokenKind()
	left, handled := p.handleNUD(kind)
	if !handled {
		p.unknownTokenErr()
		return &ast.BadExpression{Token: kind}
	}
	for BindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleLED(kind, left, BindingPowerMap[p.CurrentTokenKind()])
		if !handled {
			p.unknownTokenErr()
			continue
		}
	}
	// left = left.SetPos(left.GetRange().Start, p.savePos())
	return left
}

func (p *Parser) ParseTopLevelStatement() ast.Statement {
	kind := p.CurrentTokenKind()
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
	kind := p.CurrentTokenKind()
	result, handled := p.handleStatement(kind, false)
	if handled {
		p.Expect(lexer.EndOfStatement)
		return result
	}
	res := p.ParseLED(DefaultBindingPower)
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

func parseSeriesWithBP[T any](
	p *Parser, arr *[]T,
	bp BindingPower, until, sepBy lexer.TokenType,
) {
	parseSeries(
		p, arr,
		func() T { return p.ParseExpression(bp).(T) },
		until, sepBy, false,
	)
}

func parseSeries[T any](
	p *Parser, arr *[]T,
	with func() T, until, sepBy lexer.TokenType,
	end bool,
) {
	parse := func() T {
		start := p.CurrentToken().Position
		item := with()
		if n, ok := any(item).(ast.Node); ok && n.GetRange().IsZero() {
			item = markStartEndPos(p, n, start).(T)
		}
		return item
	}
	if end {
		for p.WhileNot(until) {
			*arr = append(*arr, parse())
			if sepBy != 0 && p.CurrentTokenKind() != until {
				p.Expect(sepBy)
			}
		}
	} else {
		for p.WhileNotEndOr(until) {
			*arr = append(*arr, parse())
			if sepBy != 0 && p.IsNotCurrentlyEndOr(until) {
				p.Expect(sepBy)
			}
		}
	}
	p.Expect(until)
}
