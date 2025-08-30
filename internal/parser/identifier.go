package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

var validIdents = map[lexer.TokenType]struct{}{
	lexer.Identifier: {},
}

func init() {
	for _, tok := range ast.Modifiers {
		validIdents[tok] = struct{}{}
	}
}

// isValidIdentifier reports whether tok is a valid identifier. Valid identifiers are
// [lexer.Identifier] and types in [ast.Modifiers].
func isValidIdentifier(tok lexer.TokenType) bool {
	return tok == lexer.Identifier || slices.Contains(ast.Modifiers, tok)
}

func isValidIdentOrDiscard(tok lexer.TokenType) bool {
	return tok == lexer.Identifier ||
		tok == lexer.Underscore || slices.Contains(ast.Modifiers, tok)
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

func (p *Parser) ParseIdentWithDiscard() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	if tok.Kind != lexer.Underscore {
		p.validateIdentifier(tok)
	}
	return ast.Identifier{Name: tok.Source, Position: tok.Position}
}

func (p *Parser) ParseMapIdentifier(includingNumber bool) ast.Identifier {
	tok := p.AdvanceNonBoundary()
	kind := tok.Kind
	switch {
	case kind == lexer.Identifier:
		break
	case kind == lexer.Numeric && !includingNumber:
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
