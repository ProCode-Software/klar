package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) validatePublic() {
	if p.CurrKind() == lexer.Public {
		p.Error(klarerrs.Token(klarerrs.ErrPublicGoesFirst, p.Curr()))
	}
}

func (p *Parser) ParsePublicModifier() ast.Statement {
	firstPublic := p.Expect(lexer.Public)
	var stmt ast.Statement
	switch curr := p.Curr(); curr.Kind {
	case lexer.Public:
		err := klarerrs.Token(klarerrs.ErrDuplicateModifier, curr)
		err.SetParam("modifier", lexer.Public)
		// Show where it was already used
		err.Highlights = append(err.Highlights, klarerrs.Highlight{
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
	case *ast.FunctionDeclaration, ast.TypeDeclaration,
		*ast.VariableDeclaration, *ast.FuncAliasDeclaration,
		ast.ModifierDeclaration:
		return &ast.PublicDeclaration{Declaration: stmt}
	default:
		p.Error(klarerrs.Node(klarerrs.ErrInvalidPublic, stmt))
	}
	return &ast.BadExpression{Value: stmt}
}
