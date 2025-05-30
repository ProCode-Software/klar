package parser

import (
	"log"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Parse parses tokens into a Program. If continueOnErr is true, the parser will
// not stop parsing on a syntax error.
func (p *Parser) Parse() (program ast.Program) {
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
			return
		}
		body = append(body, p.ParseTopLevelStatement())
	}
	return ast.Program{Body: body, Comments: comments}
}

func (p *Parser) unknownTokenErr(advance bool) {
	p.Error(errors.UnexpectedToken(p.CurrentToken()))
	if advance {
		p.Advance()
	}
}

func (p *Parser) ParseExpression(bp BindingPower) ast.Expression {
	expr := p.ParseLED(bp)
	if _, ok := expr.(ast.Expression); !ok {
		p.Error(errors.ParseError{
			Type: errors.ErrExpectedExpression,
			Node: expr,
		})
		return ast.BadExpression{Value: expr}
	}
	return expr.(ast.Expression)
}

func (p *Parser) ParseLED(bp BindingPower) ast.Node {
	kind := p.CurrentTokenKind()
	left, handled := p.handleNUD(kind)
	if !handled {
		p.unknownTokenErr(false)
		return ast.BadExpression{}
	}
	for BindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleLED(kind, left, BindingPowerMap[p.CurrentTokenKind()])
		if !handled {
			p.unknownTokenErr(true)
			continue
		}
	}
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
		return ast.ExpressionStatement{Expression: res}
	// I don't know what this is. If this occurs, then it is a bug.
	default:
		log.Panicf("node %v is neither an expression nor statement", res)
		return nil
	}
}
