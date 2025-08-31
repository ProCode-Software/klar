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
	File    string
	Errors  []ParseError

	// Conditional flags
	isWhenGuard bool // Disable some types of expressions
	isWhenCase  bool // Allow '_'
	isAttribute bool
}

type ParseError = errors.ParseError

type ParseOptions struct {
	File        string
	StopOnError bool
	OnError     func(e ParseError)
	MaxErrors   int // Parsing is stopped once len(p.Errors) equals this number. If set to 0 or less, there can be unlimited errors
}

// New returns a new [Parser] that reads from tokens.
func New(tokens []lexer.Token, options *ParseOptions) *Parser {
	t := make([]lexer.Token, len(tokens))
	copy(t, tokens)
	// Add EOS if missing
	if len(t) > 0 && t[len(t)-1].Kind != lexer.EOF {
		t = append(t, lexer.Token{Kind: lexer.EOF}) // Add position info?
	}
	return &Parser{
		Tokens:  t,
		Index:   0,
		File:    options.File,
		Options: *options,
	}
}

// Curr return the [lexer.Token] at the current parser index.
func (p *Parser) Curr() lexer.Token {
	return p.Tokens[p.Index]
}

// PeekBehind return the [lexer.Token] before the current parser index.
func (p *Parser) PeekBehind() lexer.Token {
	return p.Tokens[p.Index-1]
}

// Peek return the next [lexer.Token] without advancing the parser.
func (p *Parser) Peek() lexer.Token {
	if !p.HasTokens() {
		return p.Curr()
	}
	return p.Tokens[p.Index+1]
}

// CurrKind return the Kind of the [lexer.Token] at the current parser index.
func (p *Parser) CurrKind() lexer.TokenType {
	return p.Curr().Kind
}

// Backup decrements the parser's index by 1.
func (p *Parser) Backup() {
	if p.Index > 0 {
		p.Index--
	}
}

// Advance returns the current Token and increases the parser index.
func (p *Parser) Advance() lexer.Token {
	tok := p.Curr()
	if tok.Kind == lexer.EOF {
		return tok
	}
	p.Index++
	return tok
}

// HasTokens reports whether the parser is not at EOF.
func (p *Parser) HasTokens() bool {
	return p.Index < len(p.Tokens) && p.CurrKind() != lexer.EOF
}

// Expect advances the parser if the current token is of typ, otherwise throws an
// ExpectedTokenError.
func (p *Parser) Expect(need ...lexer.TokenType) lexer.Token {
	return p.ExpectError(nil, need...)
}

// WhileNot reports whether the current token kind is not kind and the parser is not at EOF.
func (p *Parser) WhileNot(kind lexer.TokenType) bool {
	return p.HasTokens() && p.CurrKind() != kind
}

// WhileNotEndOr reports whether the current token kind is not kind and the parser is
// not at EOF or EOS.
func (p *Parser) WhileNotEndOr(kind lexer.TokenType) bool {
	return p.HasTokens() &&
		p.CurrKind() != lexer.EndOfStatement &&
		p.CurrKind() != kind
}

// IsCurrently reports whether the current token is one of kinds.
func (p *Parser) IsCurrently(kinds ...lexer.TokenType) bool {
	return slices.Contains(kinds, p.CurrKind())
}

// IsNotCurrentlyEndOr returns true if the current token is not EOF, EOS. or kind.
func (p *Parser) IsNotCurrentlyEndOr(kind lexer.TokenType) bool {
	curr := p.CurrKind()
	return p.HasTokens() && curr != lexer.EndOfStatement && curr != kind
}

// ExpectErrorCode adds a [errors.ParseError] with code to the parser if it the
// current token is not in need.
func (p *Parser) ExpectErrorCode(code errors.ErrorCode, need ...lexer.TokenType) lexer.Token {
	return p.ExpectError(ParseError{ErrorCode: code}, need...)
}

// Expect advances the parser if the current token is of typ, otherwise throws err.
func (p *Parser) ExpectError(err error, need ...lexer.TokenType) lexer.Token {
	token := p.Curr()
	got := token.Kind
	if !slices.Contains(need, got) {
		parseErr, _ := err.(ParseError)
		if err == nil {
			err = errors.ExpectedToken(need[0], token)
		} else if parseErr.Token.Kind == 0 {
			parseErr.Token = token
			err = parseErr
		}
		p.Error(err.(ParseError))
	}
	if got == lexer.EOF {
		return token // Avoid advancing
	}
	return p.Advance()
}

// RemoveComments removes all comments from p.Tokens and returns them into a new slice.
// Errors are reported to the parser if block comments are unterminated or shebangs
// are not on the first line. These are the first errors reported in the parsing process.
func (p *Parser) RemoveComments() (comments []*ast.Comment) {
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
			comments = append(comments, &ast.Comment{
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

type stopParsing struct{}

// Error adds an error to the parser.
func (p *Parser) Error(err errors.ParseError) {
	err.File = p.File
	p.Errors = append(p.Errors, err)
	if p.Options.OnError != nil {
		p.Options.OnError(err)
	}
	if p.Options.StopOnError ||
		(p.Options.MaxErrors > 0 && len(p.Errors) >= p.Options.MaxErrors) {
		panic(stopParsing{})
	}
}

// AdvanceNonBoundary returns the current token advances the parser and returns the current token
// if it is not a boundary, otherwise returns the current token. Useful when an expected
// token is missing.
func (p *Parser) AdvanceNonBoundary() lexer.Token {
	c := p.Curr()
	switch c.Kind {
	case lexer.EndOfStatement, lexer.EOF, lexer.RightCurlyBrace,
		lexer.RightParenthesis, lexer.RightBracket, lexer.Comma:
	default:
		p.Advance()
	}
	return c
}
