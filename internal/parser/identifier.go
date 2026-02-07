package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// isValidIdentifier reports whether tok's Kind is [lexer.Identifier].
func isValidIdentifier(t lexer.TokenType) bool {
	return t == lexer.Identifier
}

func isValidIdentOrDiscard(t lexer.TokenType) bool {
	return isValidIdentifier(t) || t == lexer.Underscore
}

// validateIdentifier reports whether tok is a valid identifier. If it is false,
// validateIdentifier reports an error to the parser.
func (p *Parser) validateIdentifier(tok lexer.Token) bool {
	if isValidIdentifier(tok.Kind) {
		return true
	}
	switch {
	case tok.Kind == lexer.Underscore:
		p.Error(errors.Token(errors.ErrUnderscoreValue, tok))
	case slices.Contains(ast.ReservedIdent, tok.Kind):
		p.Error(errors.Token(errors.ErrReservedKeyword, tok))
	default:
		p.Error(errors.ExpectedToken(lexer.Identifier, tok))
	}
	return false
}

func newIdentifier(t lexer.Token) ast.Identifier {
	return ast.Identifier{Name: t.Source, Position: t.Position, Len: t.Len()}
}

func (p *Parser) ParseIdentifier() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	p.validateIdentifier(tok)
	return newIdentifier(tok)
}

func (p *Parser) ParseStrictIdentifier() ast.Identifier {
	tok := p.Expect(lexer.Identifier)
	return newIdentifier(tok)
}

// [*Parser.ParseIdentifier] but will not validate. Expected use case if for already
// validated identifiers. [lexer.Underscore] is allowed (because any token is allowed)
func (p *Parser) ParseValidIdent() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	return newIdentifier(tok)
}

func (p *Parser) ParseIdentOrDiscard() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	if tok.Kind != lexer.Underscore {
		p.validateIdentifier(tok)
	}
	return newIdentifier(tok)
}

const includingNumber, isLabel uint8 = 1, 2

// opt1: includingNumber, opt2: isLabel (for a better error)
func (p *Parser) ParseMapIdentifier(opts uint8) ast.Identifier {
	tok := p.AdvanceNonBoundary()
	kind := tok.Kind
	switch {
	case kind == lexer.Identifier:
		break
	case kind == lexer.Numeric && opts&includingNumber == 0:
		if opts&isLabel != 0 {
			p.Error(errors.Token(errors.ErrNumericLabel, tok))
			break
		}
		fallthrough
	case !slices.Contains(ast.Modifiers, tok.Kind) &&
		!slices.Contains(ast.ReservedIdent, tok.Kind):
		p.Error(errors.ExpectedToken(lexer.Identifier, tok))
	}
	return newIdentifier(tok)
}

func symbolToIdentifier(s *ast.Symbol) ast.Identifier {
	return ast.Identifier{
		Name:     s.Identifier,
		Position: s.Range.Start,
		Len:      s.Range.LineLength(),
	}
}
