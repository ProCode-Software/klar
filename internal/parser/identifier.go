package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

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
	if tok.Kind == lexer.Identifier {
		return true
	}
	if _, ok := validIdents[tok.Kind]; !ok {
		if slices.Contains(ast.ReservedIdent, tok.Kind) {
			p.Error(errors.Token(errors.ErrReservedKeyword, tok))
		} else {
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

// opt1: includingNumber, opt2: isLabel (for a better error)
func (p *Parser) ParseMapIdentifier(opts ...bool) ast.Identifier {
	var (
		includingNumber = len(opts) > 0 && opts[0]
		isLabel         = len(opts) > 1 && opts[1]
		tok             = p.AdvanceNonBoundary()
		kind            = tok.Kind
	)
	switch {
	case kind == lexer.Identifier:
		break
	case kind == lexer.Numeric && !includingNumber:
		if isLabel {
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
