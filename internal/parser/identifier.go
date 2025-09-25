package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Valid unless explicitly parsed
var validIdents = map[lexer.TokenType]struct{}{
	lexer.Identifier: {}, lexer.Import: {}, lexer.Can: {},
}

// isValidIdentifier reports whether tok is a valid identifier. Valid identifiers are
// [lexer.Identifier] and types in [ast.Modifiers].
func isValidIdentifier(tok lexer.TokenType) bool {
	_, valid := validIdents[tok]
	return valid
}

func isValidIdentOrDiscard(tok lexer.TokenType) bool {
	_, valid := validIdents[tok]
	return valid || tok == lexer.Underscore
}

// validateIdentifier reports whether tok is a valid identifier. If it is false,
// validateIdentifier reports an error to the parser.
func (p *Parser) validateIdentifier(tok lexer.Token) bool {
	if _, ok := validIdents[tok.Kind]; !ok {
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
	return true
}

func (p *Parser) ParseIdentifier() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	p.validateIdentifier(tok)
	return ast.Identifier{Name: tok.Source, Position: tok.Position}
}

// [*Parser.ParseIdentifier] but will not validate. Expected use case if for already
// validated identifiers. [lexer.Underscore] is allowed (because any token is allowed)
func (p *Parser) ParseValidIdent() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	return ast.Identifier{Name: tok.Source, Position: tok.Position}
}

func (p *Parser) ParseIdentOrDiscard() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	if tok.Kind != lexer.Underscore {
		p.validateIdentifier(tok)
	}
	return ast.Identifier{Name: tok.Source, Position: tok.Position}
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
			p.Error(errors.Token(errors.ErrInvalidLabel, tok))
			break
		}
		fallthrough
	case !slices.Contains(ast.Modifiers, tok.Kind) &&
		!slices.Contains(ast.ReservedIdent, tok.Kind):
		p.Error(errors.ExpectedToken(lexer.Identifier, tok))
	}
	return ast.Identifier{Name: tok.Source, Position: tok.Position}
}

func symbolToIdentifier(s *ast.Symbol) ast.Identifier {
	return ast.Identifier{Name: s.Identifier, Position: s.Range.Start}
}
