// Package parser implements a parser that converts [lexer.Token] into an AST.
package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
)

const (
	isAttribute = 1 << iota // Allow version parsing
	whenPattern             // Allow '_'
)

// A Parser parses lexer tokens into an abstract syntax tree (AST).
type Parser struct {
	Options        Options
	Tokens         []lexer.Token
	Index          int
	Errors         []*Error
	flags          uint8            // Conditional flags
	listCastTokens map[int]struct{} // Populated during EOS inference
}

// Error is [klarerrs.Error]
type Error = klarerrs.Error

// Options is options provided to [Parser].
type Options struct {
	// Path of the file being parsed. File is applied to all reported errors.
	File string
	// If Error != nil, Error is called for every reported error.
	Error func(e *Error)
	// Parsing is stopped once len(p.Errors) equals MaxErrors.
	// If MaxErrors <= 0, there is no limit.
	MaxErrors int
}

// New returns a new [Parser] that reads from tokens. If options == nil,
// default options are used.
func New(tokens []lexer.Token, options *Options) *Parser {
	if options == nil {
		options = &Options{}
	}
	return &Parser{
		Tokens:  tokens,
		Index:   0,
		Options: *options,
	}
}

// Curr return the [lexer.Token] at the current parser index.
func (p *Parser) Curr() lexer.Token { return p.Tokens[p.Index] }

// PeekBehind return the [lexer.Token] before the current parser index.
func (p *Parser) PeekBehind() lexer.Token { return p.Tokens[p.Index-1] }

// Peek return the next [lexer.Token] without advancing the parser.
func (p *Parser) Peek() lexer.Token {
	if !p.HasTokens() {
		return p.Curr()
	}
	return p.Tokens[p.Index+1]
}

func (p *Parser) PeekKind() lexer.TokenType { return p.Peek().Kind }

// CurrKind return the Kind of the [lexer.Token] at the current parser index.
func (p *Parser) CurrKind() lexer.TokenType { return p.Curr().Kind }

// Backup decrements the parser's index by 1.
func (p *Parser) Backup() {
	if p.Index <= 0 {
		panic("Parser.Backup() called with Index 0")
	}
	p.Index--
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

// WhileNot reports whether the current token kind is not kind and the parser is not at EOF.
func (p *Parser) WhileNot(kind lexer.TokenType) bool {
	return p.HasTokens() && p.CurrKind() != kind
}

// WhileNotEndOr reports whether the current token kind is not kind and the parser is
// not at EOF or EOS.
func (p *Parser) WhileNotEndOr(kind lexer.TokenType) bool {
	return p.HasTokens() &&
		p.CurrKind() != lexer.Newline &&
		p.CurrKind() != kind
}

// IsCurrently reports whether the current token is one of kinds.
func (p *Parser) IsCurrently(kinds ...lexer.TokenType) bool {
	return slices.Contains(kinds, p.CurrKind())
}

// IsNotCurrentlyEndOr returns true if the current token is not EOF, EOS. or kind.
func (p *Parser) IsNotCurrentlyEndOr(kind lexer.TokenType) bool {
	curr := p.CurrKind()
	return p.HasTokens() && curr != lexer.Newline && curr != kind
}

// Expect advances the parser if the current token is of typ, otherwise throws an
// ExpectedTokenError.
func (p *Parser) Expect(exp lexer.TokenType, expFlags ...expectFlag) lexer.Token {
	got := p.Curr()
	if got.Kind == exp {
		return p.Advance()
	}
	err, noAdvance := withExpectFlags(expFlags, exp, got)
	p.Error(err)
	if noAdvance {
		return got
	}
	return p.Advance()
}

func (p *Parser) ExpectOneOf(a, b lexer.TokenType, expFlags ...expectFlag) lexer.Token {
	got := p.Curr()
	if got.Kind == a || got.Kind == b {
		return p.Advance()
	}
	err, noAdvance := withExpectFlags(expFlags, a, got)
	p.Error(err)
	if noAdvance {
		return got
	}
	return p.Advance()
}

// If stopParsing is passed to panic, the parser will immediately stop parsing.
type stopParsing struct{}

type stmtError struct{}

// Error adds an error to the parser.
func (p *Parser) Error(err *klarerrs.Error) {
	err.File = p.Options.File
	p.Errors = append(p.Errors, err)
	if p.Options.Error != nil {
		p.Options.Error(err)
	}
	if mx := p.Options.MaxErrors; mx > 0 && len(p.Errors) >= mx {
		p.Errors = append(p.Errors, klarerrs.TooManyErrors())
		panic(stopParsing{})
	}
}

func (p *Parser) ErrorLabelled(err *klarerrs.Error, label string) {
	err.Label = label
	p.Error(err)
}

// AdvanceNonBoundary returns the current token advances the parser and returns
// the current token if it is not a boundary, otherwise returns the current token.
// Useful when an expected token is missing.
func (p *Parser) AdvanceNonBoundary() lexer.Token {
	c := p.Curr()
	switch c.Kind {
	case lexer.Newline, lexer.EOF, lexer.RightCurlyBrace,
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

func (p *Parser) isWhenPattern() bool { return (p.flags & whenPattern) != 0 }
func (p *Parser) isAttribute() bool   { return (p.flags & isAttribute) != 0 }

// Reset resets all properties to defaults, freeing resources.
func (p *Parser) Reset() {
	p.Options.Error = nil
	p.Options.File = ""
	p.Options.MaxErrors = 0

	p.Tokens = nil
	p.Index = 0
	p.Errors = nil
	p.flags = 0

	p.listCastTokens = nil
}
