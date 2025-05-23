package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseTypeDeclaration() ast.TypeDeclaration {
	p.Expect(lexer.Type)
	var (
		name = p.Expect(lexer.Identifier)
		/* inherited string
		isEnum bool
		fields []any */
	)

	switch p.Advance().Kind {
	case lexer.Equal:
		// Type
		return ast.TypeAliasDeclaration{
			Identifier: name.Source,
			Type:       p.ParseType(DefaultBindingPower, false),
		}
	case lexer.Colon:
		// Inherited struct
		_ = p.Expect(lexer.Identifier).Source
		fallthrough
	case lexer.LeftCurlyBrace:
		// Struct or enum
		for p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Advance()
		}
		p.Expect(lexer.RightCurlyBrace)
	default:
		// Some other token or unassigned type (if EOS)
		panic(errors.NewTokenError(errors.ErrExpectedTypeAssignment, p.CurrentToken()))
	}
	return nil
}
