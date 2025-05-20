package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Parse parses tokens into a Program. If continueOnErr is true, the parser will
// not panic on a syntax error.
func Parse(tokens []lexer.Token, continueOnErr bool) (program ast.Program, errs []error) {
	var (
		shouldBreak bool
		body        = make([]ast.ASTItem, 0, len(tokens)/2)
		p           = New(tokens)
		comments    = p.RemoveComments() // Move comments
	)
	p.InsertEOS() // Add the "semicolons"
	for p.HasTokens() && !shouldBreak {
		func() {
			defer func() {
				if err := recover(); err != nil {
					_, isParseErr := err.(errors.ParseError)
					switch {
					case isParseErr:
						errs = append(errs, err.(errors.ParseError))
						p.Index++
					case !isParseErr:
						errs = append(errs, err.(error))
						shouldBreak = true
					}
					if !continueOnErr {
						shouldBreak = true
					}
				}
			}()
			body = append(body, p.ParseStatement())
		}()
	}
	return ast.Program{Body: body, Comments: comments}, errs
}

// For debugging purposes
func noHandlerError(p *Parser, nudOrLED string) {
	panic(fmt.Sprintf(
		"Unexpected token '%s' (expected %s handler for %s)\n",
		p.CurrentToken().Source,
		nudOrLED,
		lexer.TokenTypes[p.CurrentTokenKind()],
	))
}

func (p *Parser) unknownTokenErr() {
	panic(errors.UnknownTokenError(p.CurrentToken()))
}

func (p *Parser) ParseExpression(bp BindingPower) ast.Expression {
	expr := p.ParseLED(bp)
	if _, ok := expr.(ast.Expression); !ok {
		panic(errors.ParseError{
			Type:    errors.ErrExpectedExpression,
			ASTItem: expr,
		})
	}
	return expr.(ast.Expression)
}

func (p *Parser) ParseLED(bp BindingPower) ast.ASTItem {
	kind := p.CurrentTokenKind()
	left, handled := p.handleNUD(kind)
	if !handled {
		p.unknownTokenErr()
	}
	for BindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleLED(kind, left, bp)
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
		return result
	}

	res := p.ParseLED(DefaultBindingPower)
	p.Expect(lexer.EndOfStatement)
	switch res.(type) {
	// Left-denoted statement
	case ast.Statement:
		return res.(ast.Statement)

	// Then it is an expression
	case ast.Expression:
		return ast.ExpressionStatement{Expression: res.(ast.Expression)}

	// I don't know what this is
	default:
		return nil
	}
}
