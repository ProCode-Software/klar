// Package parser implements a parser that converts [lexer.Token] into an AST.
package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// A Parser parses lexer tokens into an abstract syntax tree (AST).
type Parser struct {
	Options ParseOptions
	Tokens  []lexer.Token
	Index   int
	Errors  []ParseError

	// Conditional flags
	isWhenGuard bool // Disable some types of expressions
	isWhenCase  bool // Allow '_'
}

// New returns a new [Parser] that reads from tokens.
func New(tokens []lexer.Token) *Parser {
	t := make([]lexer.Token, len(tokens))
	copy(t, tokens)
	return &Parser{
		Tokens: t,
		Index:  0,
	}
}

// CurrentToken return the [lexer.Token] at the current parser index.
func (p *Parser) CurrentToken() lexer.Token {
	return p.Tokens[p.Index]
}

// PeekBehind return the [lexer.Token] before the current parser index.
func (p *Parser) PeekBehind() lexer.Token {
	return p.Tokens[p.Index-1]
}

// Peek return the next [lexer.Token].
func (p *Parser) Peek() lexer.Token {
	if !p.HasTokens() {
		return p.CurrentToken()
	}
	return p.Tokens[p.Index+1]
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
	if tok.Kind == lexer.EOF {
		return tok
	}
	p.Index++
	return tok
}

// HasTokens reports whether the parser is not at EOF.
func (p *Parser) HasTokens() bool {
	return p.Index < len(p.Tokens) && p.CurrentTokenKind() != lexer.EOF
}

// Expect advances the parser if the current token is of typ, otherwise throws an
// ExpectedTokenError.
func (p *Parser) Expect(need ...lexer.TokenType) lexer.Token {
	return p.ExpectError(nil, need...)
}

// WhileNot reports whether the current token kind is not kind and the parser is not at EOF.
func (p *Parser) WhileNot(kind lexer.TokenType) bool {
	return p.HasTokens() && p.CurrentTokenKind() != kind
}

// WhileNotEndOr reports whether the current token kind is not kind and the parser is
// not at EOF or EOS.
func (p *Parser) WhileNotEndOr(kind lexer.TokenType) bool {
	return p.HasTokens() &&
		p.CurrentTokenKind() != lexer.EndOfStatement &&
		p.CurrentTokenKind() != kind
}

// IsCurrently reports whether the current token is one of kinds.
func (p *Parser) IsCurrently(kinds ...lexer.TokenType) bool {
	return slices.Contains(kinds, p.CurrentTokenKind())
}

// IsNotCurrentlyEndOr returns true if the current token is not EOF, EOS. or kind.
func (p *Parser) IsNotCurrentlyEndOr(kind lexer.TokenType) bool {
	curr := p.CurrentTokenKind()
	return p.HasTokens() && curr != lexer.EndOfStatement && curr != kind
}

func (p *Parser) isMapIdentifier() bool {
	return p.IsCurrently(ast.ReservedIdent...) ||
		p.IsCurrently(lexer.Identifier, lexer.Numeric, lexer.Boolean, lexer.Nil)
}

func (p *Parser) expectMapIdent() lexer.Token {
	if !p.isMapIdentifier() {
		return p.Expect(lexer.Identifier)
	}
	return p.Advance()
}

func (p *Parser) expectNonNumericMapIdent() lexer.Token {
	if !p.isMapIdentifier() || p.CurrentTokenKind() == lexer.Numeric {
		return p.Expect(lexer.Identifier)
	}
	return p.Advance()
}

func (p *Parser) lastTokEnd() lexer.Position {
	last := p.Tokens[p.Index-1]
	return ranges.FromToken(last).End
}

func (p *Parser) expectShorthand() (key ast.Symbol, value ast.Expression) {
	sym, isOk := p.ParseExpression(CallBindingPower), false
	switch sym := sym.(type) {
	case ast.Symbol:
		key = sym
		value = sym
		isOk = true
	case ast.IndexExpression:
		if sym.Computed {
			break
		}
		if prop, ok := sym.Property.(ast.Symbol); ok {
			key = prop
			value = sym
			isOk = true
		}
	}
	if !isOk {
		p.Error(errors.Node(errors.ErrInvalidLabelShorthand, sym))
	}
	return key, value
}

func copyPos[S, T ast.Node](from S, to T) T {
	return to.SetPos(from.Base().Start, from.Base().End).(T)
}

// Expect advances the parser if the current token is of typ, otherwise throws err.
func (p *Parser) ExpectError(err error, need ...lexer.TokenType) lexer.Token {
	token := p.CurrentToken()
	got := token.Kind
	if !slices.Contains(need, got) {
		if err == nil {
			err = errors.ExpectedToken(need[0], token)
		}
		p.Error(err.(ParseError))
	}
	if got == lexer.EOF {
		return token // Avoid advancing
	}
	return p.Advance()
}

func (p *Parser) savePos() lexer.Position {
	return p.CurrentToken().Position
}

// RemoveComments removes all comments from p.Tokens and returns them into a new slice.
func (p *Parser) RemoveComments() (comments []ast.Comment) {
	for i := 0; i < len(p.Tokens); i++ {
		tok := p.Tokens[i]
		switch tok.Kind {
		case lexer.BlockComment, lexer.LineComment, lexer.Hashbang:
			switch {
			case tok.Kind == lexer.Hashbang:
				if tok.Position != (lexer.Position{1, 1}) {
					p.Error(errors.Token(errors.ErrMisplacedShebang, tok))
				}
			case tok.Attributes["unterm"] == true:
				p.Error(errors.ParseError{
					ErrorCode: errors.ErrUnterminatedComment,
					Token:     tok,
					Position:  tok.Position,
				})
			}
			comments = append(comments, ast.Comment{
				Value:    tok.Source,
				Type:     tok.Kind,
				BaseNode: ast.BaseNode{ranges.FromToken(tok)},
			})
			p.Tokens = slices.Delete(p.Tokens, i, i+1)
			i--
		}
	}
	return comments
}

func (p *Parser) Error(err errors.ParseError) {
	p.Errors = append(p.Errors, err)
	if p.Options.OnError != nil {
		p.Options.OnError(err)
	}
}
