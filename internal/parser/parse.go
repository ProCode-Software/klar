package parser

import (
	"fmt"
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Parse parses tokens into a Program. If continueOnErr is true, the parser will
// not panic on a syntax error.
func Parse(tokens []lexer.Token, continueOnErr bool) (program ast.Program, errs []error) {
	var (
		shouldBreak bool
		body        = make([]ast.Statement, 0, len(tokens)/2)
		p           = New(tokens)
		comments    = p.RemoveComments() // Move comments
	)
	p.InsertEOS() // Add the "semicolons"
	for p.HasTokens() && !shouldBreak {
		func() {
			defer p.handleError(continueOnErr, &errs, &shouldBreak)
			body = append(body, p.ParseStatement())
		}()
	}
	return ast.Program{Body: body, Comments: comments}, errs
}

func (p *Parser) handleError(continueOnErr bool, errs *[]error, shouldBreak *bool) {
	unshift := func(err error) {
		*errs = slices.Insert(*errs, 0, err)
	}
	if err := recover(); err != nil {
		switch err := err.(type) {
		case errors.ParseError:
			unshift(err)
			if !continueOnErr {
				*shouldBreak = true
				return
			}
			p.Index++
		case error:
			unshift(err)
			*shouldBreak = true
		default:
			unshift(fmt.Errorf("%v", err))
			*shouldBreak = true
		}
	}
}

// For debugging purposes
func noHandlerError(p *Parser, nudOrLED string) {
	panic(fmt.Sprintf(
		"Unexpected token '%s' (need %s handler for %s)\n",
		p.CurrentToken().Source,
		nudOrLED,
		p.CurrentTokenKind().String(),
	))
}

func (p *Parser) unknownTokenErr() {
	panic(errors.UnexpectedTokenError(p.CurrentToken()))
}

func (p *Parser) ParseExpression(bp BindingPower) ast.Expression {
	expr := p.ParseLED(bp)
	if _, ok := expr.(ast.Expression); !ok {
		panic(errors.ParseError{
			Type: errors.ErrExpectedExpression,
			Node: expr,
		})
	}
	return expr.(ast.Expression)
}

func (p *Parser) ParseLED(bp BindingPower) ast.Node {
	kind := p.CurrentTokenKind()
	left, handled := p.handleNUD(kind)
	if !handled {
		p.unknownTokenErr()
	}
	for BindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleLED(kind, left, BindingPowerMap[p.CurrentTokenKind()])
		if !handled {
			p.unknownTokenErr()
		}
	}
	return left
}

func (p *Parser) ParseStatement() ast.Statement {
	kind := p.CurrentTokenKind()
	result, handled := p.handleStatement(kind)
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
		return ast.ExpressionStatement{Expression: res}

	// I don't know what this is
	default:
		return nil
	}
}
