// Package parser implements a parser that converts [lexer.Token] into an AST.
package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	isAttribute = 1 << iota // Allow version parsing
	isWhenCase              // Allow '_'
)

// A Parser parses lexer tokens into an abstract syntax tree (AST).
type Parser struct {
	Options ParseOptions
	Tokens  []lexer.Token
	Index   int
	Errors  []*ParseError
	flags   uint8 // Conditional flags

	// Stored token properties
	listCastTokens, assignmentTokens map[int]struct{}
}

// ParseError is [errors.ParseError]
type ParseError = errors.ParseError

// ParseOptions is options provided to [Parser].
type ParseOptions struct {
	// Path of the file being parsed. File is applied to all reported errors.
	File string
	// If Error != nil, Error is called for every reported error.
	Error func(e *ParseError)
	// Parsing is stopped once len(p.Errors) equals MaxErrors.
	// If MaxErrors <= 0, there is no limit.
	MaxErrors int
}

// New returns a new [Parser] that reads from tokens. If options == nil,
// default options are used.
func New(tokens []lexer.Token, options *ParseOptions) *Parser {
	if options == nil {
		options = &ParseOptions{}
	}
	return &Parser{
		Tokens:  tokens,
		Index:   0,
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

func (p *Parser) PeekKind() lexer.TokenType {
	return p.Peek().Kind
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

func (p *Parser) ExpectNoAdvance(need ...lexer.TokenType) lexer.Token {
	return p.ExpectErrorNoAdvance(nil, need...)
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
	return p.ExpectError(&ParseError{ErrorCode: code}, need...)
}

func (p *Parser) ExpectErrorNoAdvance(err error, need ...lexer.TokenType) lexer.Token {
	token := p.Curr()
	got := token.Kind
	if !slices.Contains(need, got) {
		err, _ := err.(*ParseError)
		if err == nil {
			err = errors.ExpectedToken(need[0], token)
		} else if err.Token.Kind == 0 {
			err.Token = token
		}
		p.Error(err)
		return token
	}
	return p.Advance()
}

// Expect advances the parser if the current token is of typ, otherwise throws err.
func (p *Parser) ExpectError(err error, need ...lexer.TokenType) lexer.Token {
	token := p.Curr()
	got := token.Kind
	if !slices.Contains(need, got) {
		parseErr, _ := err.(*ParseError)
		if err == nil {
			err = errors.ExpectedToken(need[0], token)
		} else if parseErr.Token.Kind == 0 {
			parseErr.Token = token
			parseErr.Range = ranges.FromToken(token)
			err = parseErr
		}
		p.Error(err.(*ParseError))
	}
	if got == lexer.EOF {
		return token // Avoid advancing
	}
	return p.Advance()
}

// If stopParsing is passed to panic, the parser will immediately stop parsing.
type stopParsing struct{}

// Error adds an error to the parser.
func (p *Parser) Error(err *errors.ParseError) {
	err.File = p.Options.File
	p.Errors = append(p.Errors, err)
	if p.Options.Error != nil {
		p.Options.Error(err)
	}
	if mx := p.Options.MaxErrors; mx > 0 && len(p.Errors) >= mx {
		p.Errors = append(p.Errors, errors.TooManyErrors())
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

func (p *Parser) handlePanic() {
	switch err := recover(); err.(type) {
	case nil, stopParsing: // Bailout
		return
	default:
		panic(err) // Re-panic if other error
	}
}

func (p *Parser) isWhenCase() bool  { return (p.flags & isWhenCase) != 0 }
func (p *Parser) isAttribute() bool { return (p.flags & isAttribute) != 0 }

// Reset resets all properties to defaults, freeing resources.
func (p *Parser) Reset() {
	p.Options.Error = nil
	p.Options.File = ""
	p.Options.MaxErrors = 0

	p.Tokens = nil
	p.Index = 0
	p.Errors = nil
	p.flags = 0

	p.assignmentTokens, p.listCastTokens = nil, nil
}
