// Package parser implements a parser that converts [lexer.Token] into an AST.
package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// A Parser parses lexer tokens into an abstract syntax tree (AST)
type Parser struct {
	Tokens []lexer.Token
	Index  int
}

// New returns a new [Parser] that reads from tokens.
func New(tokens []lexer.Token) *Parser {
	return &Parser{
		Tokens: tokens,
		Index:  0,
	}
}

// CurrentToken return the [lexer.Token] at the current parser index.
func (p *Parser) CurrentToken() lexer.Token {
	return p.Tokens[p.Index]
}

// CurrentTokenKind return the Kind of the [lexer.Token] at the current parser index.
func (p *Parser) CurrentTokenKind() lexer.TokenType {
	return p.CurrentToken().Kind
}

// Backup decrements the parser's index by 1.
func (p *Parser) Backup() {
	if p.Index > 0 {
		p.Index--
	}
}

// Advance returns the current Token and increases the parser index.
func (p *Parser) Advance() lexer.Token {
	tok := p.CurrentToken()
	p.Index++
	return tok
}

// HasTokens reports whether the parser is not at EOF.
func (p *Parser) HasTokens() bool {
	return p.Index < len(p.Tokens) && p.CurrentTokenKind() != lexer.EOF
}

// Expect returns the next token if it is of typ, otherwise panics.
func (p *Parser) Expect(typ lexer.TokenType) lexer.Token {
	return p.ExpectError(typ, nil)
}

func (p *Parser) ExpectError(exp lexer.TokenType, err error) lexer.Token {
	token := p.CurrentToken()
	got := token.Kind
	if got != exp {
		if err == nil {
			err = errors.ExpectedTokenError(exp, token, token.Position)
		}
		panic(err)
	}
	return p.Advance()
}

// RemoveComments removes all comments from p.Tokens and returns them into a new slice.
func (p *Parser) RemoveComments() (comments []ast.Comment) {
	for i, tok := range p.Tokens {
		if tok.Kind == lexer.BlockComment || tok.Kind == lexer.LineComment {
			comments = append(comments, ast.Comment{
				Begin: tok.Position,
				End: lexer.Position{
					Line: tok.Position.Line,
					Col:  tok.Position.Col + len(tok.Source),
				},
				Value: tok.Source,
				Type:  ast.CommentType(tok.Kind - 6), // Line: 6, Block: 7
			})
			p.Tokens = slices.Delete(p.Tokens, i, i+1)
		}
	}
	return comments
}

func Parse(tokens []lexer.Token, continueOnErr bool) ast.Program {
	var (
		body     = make([]ast.ASTItem, 0, len(tokens)/2)
		p        = New(tokens)
		comments = p.RemoveComments() // Move comments
	)
	p.InsertEOS() // Add the "semicolons"
	for p.HasTokens() {
		func() {
			defer p.handleSyntaxError(continueOnErr)
			if p.CurrentTokenKind() == lexer.Illegal {
				panic(errors.UnknownTokenError(p.CurrentToken()))
			}
			body = append(body, p.ParseStatement())
		}()
	}
	return ast.Program{Body: body, Comments: comments}
}

func (p *Parser) handleSyntaxError(continueOnErr bool) {
	if !continueOnErr {
		return
	}
	if err := recover(); err != nil {
		if _, isParseErr := err.(errors.ParseError); isParseErr {
			p.Index++
		} else {
			// Internal error
			errors.InternalError(err)
		}
	}
}
