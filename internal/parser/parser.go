// Package parser implements a parser that converts [lexer.Token] into an AST.
package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// A Parser parses lexer tokens into an abstract syntax tree (AST).
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

// Expect advances the parser if the current token is of typ, otherwise panics.
func (p *Parser) Expect(need ...lexer.TokenType) lexer.Token {
	return p.ExpectError(nil, need...)
}

// IsNot reports whether the current token kind is not kind and the parser is not at EOF.
func (p *Parser) IsNot(kind lexer.TokenType) bool {
	return p.HasTokens() && p.CurrentTokenKind() != kind
}

// IsCurrently reports whether the current token is one of kinds.
func (p *Parser) IsCurrently(kinds ...lexer.TokenType) bool {
	return slices.Contains(kinds, p.CurrentTokenKind())
}

// Expect advances the parser if the current token is of typ, otherwise panics with err.
func (p *Parser) ExpectError(err error, need ...lexer.TokenType) lexer.Token {
	token := p.CurrentToken()
	got := token.Kind
	if !slices.Contains(need, got) {
		if err == nil {
			err = errors.ExpectedTokenError(need[0], token, token.Position)
		}
		panic(err)
	}
	return p.Advance()
}

// RemoveComments removes all comments from p.Tokens and returns them into a new slice.
func (p *Parser) RemoveComments() (comments []ast.Comment) {
	for i := 0; i < len(p.Tokens); i++ {
		tok := p.Tokens[i]
		if tok.Kind == lexer.BlockComment || tok.Kind == lexer.LineComment {
			comments = append(comments, ast.Comment{
				Begin: tok.Position,
				End: lexer.Position{
					Line: tok.Position.Line,
					Col:  tok.Position.Col + len(tok.Source),
				},
				Value: tok.Source,
				Type:  tok.Kind,
			})
			p.Tokens = slices.Delete(p.Tokens, i, i+1)
		}
	}
	return comments
}
