package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Valid unless explicitly parsed
var validIdents = map[lexer.TokenType]struct{}{
	lexer.Identifier: {}, lexer.Import: {}, lexer.Can: {},
}

// Valid unless followed by a modifier
var modifiers = map[lexer.TokenType]struct{}{
	lexer.Opaque: {}, lexer.Public: {},
}

func init() {
	for _, tok := range ast.Modifiers {
		validIdents[tok] = struct{}{}
		modifiers[tok] = struct{}{}
	}
}

// isModifierUse reports whether the current token is followed by a modifier
// or assignment
func (p *Parser) isModifierUse(_ lexer.TokenType) bool {
	if p.Index > 0 {
		if _, ok := modifiers[p.PeekBehind().Kind]; ok {
			return true
		}
	}
	nextKind := p.Peek().Kind
	if _, ok := modifiers[nextKind]; ok {
		return true
	}
	switch nextKind {
	case lexer.Identifier, lexer.Type, lexer.Func, lexer.HashLeftCurlyBrace, lexer.Underscore:
		return true
	case lexer.LeftParenthesis, lexer.LeftBracket:
		return p.Lookahead(isDestructureAssignment)
	}
	return false
}

func (p *Parser) validatePublic() {
	if p.CurrKind() == lexer.Public {
		p.Error(errors.Token(errors.ErrPublicFirst, p.Curr()))
	}
}

func (p *Parser) ParseOpaqueModifier() ast.Statement {
	p.Advance() // opaque
	p.validatePublic()
	stmt := p.ParseStatement()
	switch stmt := stmt.(type) {
	case *ast.InterfaceDeclaration, *ast.StructDeclaration:
		return &ast.OpaqueDeclaration{Declaration: stmt.(ast.TypeDeclaration)}
	case *ast.PublicDeclaration:
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
