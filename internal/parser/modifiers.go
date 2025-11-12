package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) validatePublic() {
	if p.CurrKind() == lexer.Public {
		p.Error(errors.Token(errors.ErrPublicGoesFirst, p.Curr()))
	}
}

func (p *Parser) ParseOpaqueModifier() ast.Statement {
	p.Advance() // opaque
	p.validatePublic()
	stmt := p.ParseStatement()
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
	p.Expect(lexer.Public)
	stmt := p.ParseStatement()
	switch stmt.(type) {
	case *ast.PublicDeclaration:
		err := errors.Node(errors.ErrDuplicateModifier, stmt)
		err.SetParam("modifier", lexer.Public)
		p.Error(err)
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
