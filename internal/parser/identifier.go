package parser

import (
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
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
		p.Error(klarerrs.Token(klarerrs.ErrUnderscoreValue, tok))
	case slices.Contains(lexer.ReservedIdent, tok.Kind):
		p.Error(klarerrs.Token(klarerrs.ErrReservedKeyword, tok))
	default:
		p.Error(klarerrs.ExpectedToken(lexer.Identifier, tok))
	}
	return false
}

func (p *Parser) newIdentifier(t lexer.Token) ast.Identifier {
	ident := ast.Identifier{Name: t.Source, Position: t.Position, Len: t.Len()}
	p.ValidateIdentName(ident.Name, ident)
	return ident
}

// ValidateIdentName reports an error if i's Name does not contain any letters.
// Examples of invalid identifiers: '_123', '__'
func (p *Parser) ValidateIdentName(s string, node ast.Node) {
	if s == "_" {
		return // No error
	}
	withoutUnderscore := strings.Trim(s, "_")
	var label string
	if withoutUnderscore == "" {
		// Allow discards. '___' is not allowed.
		label = "This name has only underscores"
	} else if strings.TrimFunc(withoutUnderscore, lexer.IsDigit) == "" {
		// '_123'
		label = "Identifiers should contain at least 1 letter"
	} else {
		return
	}
	err := klarerrs.Node(klarerrs.ErrIdentMustHaveLetter, node)
	err.Name = s
	err.Label = label
	p.Error(err)
}

func (p *Parser) ParseIdentifier() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	p.validateIdentifier(tok)
	return p.newIdentifier(tok)
}

func (p *Parser) ParseStrictIdentifier() ast.Identifier {
	tok := p.Expect(lexer.Identifier)
	return p.newIdentifier(tok)
}

// [*Parser.ParseIdentifier] but will not validate. Expected use case if for already
// validated identifiers. [lexer.Underscore] is allowed (because any token is allowed)
func (p *Parser) ParseValidIdent() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	return p.newIdentifier(tok)
}

func (p *Parser) ParseIdentOrDiscard() ast.Identifier {
	tok := p.AdvanceNonBoundary()
	if tok.Kind != lexer.Underscore {
		p.validateIdentifier(tok)
	}
	return p.newIdentifier(tok)
}

// opt1: includingNumber, opt2: isLabel (for a better error)
func (p *Parser) ParseMapIdentifier(opts parseFlags) ast.Identifier {
	tok := p.AdvanceNonBoundary()
	kind := tok.Kind
	switch {
	case kind == lexer.Identifier:
		break
	case kind == lexer.Numeric && opts&allowNumber == 0:
		if opts&isLabel != 0 {
			p.Error(klarerrs.Token(klarerrs.ErrNumericLabel, tok))
			break
		}
		fallthrough
	case !slices.Contains(ast.Modifiers, tok.Kind) &&
		!slices.Contains(lexer.ReservedIdent, tok.Kind):
		p.Error(klarerrs.ExpectedToken(lexer.Identifier, tok))
	}
	return p.newIdentifier(tok)
}

func (p *Parser) ParseMapIdentOrDiscard(opts parseFlags) ast.Identifier {
	if p.CurrKind() == lexer.Underscore {
		return p.ParseValidIdent()
	}
	return p.ParseMapIdentifier(opts)
}
