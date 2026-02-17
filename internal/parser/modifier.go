package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) validatePublic() {
	if p.CurrKind() == lexer.Public {
		p.Error(errors.Token(errors.ErrPublicGoesFirst, p.Curr()))
	}
}

func (p *Parser) ParseOpaqueModifier() ast.Statement {
	p.Advance() // opaque
	p.validatePublic()
	stmt := p.ParseStatement(noEOS)
	switch stmt := stmt.(type) {
	case *ast.InterfaceDeclaration, *ast.StructDeclaration:
		return &ast.OpaqueDeclaration{Declaration: stmt.(ast.TypeDeclaration)}
	case *ast.PublicDeclaration: // Already checked and invalid
	default:
		p.Error(errors.Node(errors.ErrInvalidOpaque, stmt))
	}
	return &ast.BadExpression{Value: stmt}
}

func (p *Parser) ParsePublicModifier() ast.Statement {
	firstPublic := p.Expect(lexer.Public)
	var stmt ast.Statement
	switch curr := p.Curr(); curr.Kind {
	case lexer.Opaque:
		stmt = p.ParseOpaqueModifier()
		markStartEndPos(p, stmt, curr.Position)
	case lexer.Public:
		err := errors.Token(errors.ErrDuplicateModifier, curr)
		err.SetParam("modifier", lexer.Public)
		// Show where it was already used
		err.Highlights = append(err.Highlights, errors.Highlight{
			ranges.FromToken(firstPublic), "It was already used here",
		})
		p.Error(err)
		// Still parse it
		stmt = &ast.BadExpression{Value: p.ParsePublicModifier()}
		markStartEndPos(p, stmt, curr.Position)
	default:
		stmt = p.ParseStatement(noEOS)
	}
	switch stmt.(type) {
	case *ast.BadExpression:
	case *ast.EnumDeclaration, *ast.FunctionDeclaration,
		*ast.InterfaceDeclaration, ast.TypeDeclaration,
		*ast.StructDeclaration, *ast.TypeAliasDeclaration,
		*ast.VariableDeclaration, *ast.FuncAliasDeclaration,
		ast.ModifierDeclaration:
		return &ast.PublicDeclaration{Declaration: stmt}
	default:
		p.Error(errors.Node(errors.ErrInvalidPublic, stmt))
	}
	return &ast.BadExpression{Value: stmt}
}
