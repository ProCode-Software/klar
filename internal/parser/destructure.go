package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseDestructure() ast.Destructure {
	switch p.CurrentTokenKind() {
	case lexer.HashLeftCurlyBrace:
		p.Advance()
		return p.ParseObjectDestructure()
	case lexer.Identifier, lexer.LeftBracket, lexer.LeftParenthesis:
		return p.ParseDestructureInner()
	}
	return nil
}

func (p *Parser) ParseDestructureInner() ast.Destructure {
	switch p.CurrentTokenKind() {
	case lexer.Identifier:
		return p.parseSymbolDestruct()
	case lexer.LeftCurlyBrace:
		start := p.Advance().Position
		return markStartEndPos(p, p.ParseObjectDestructure(), start)
	case lexer.LeftBracket:
		start := p.Advance().Position
		return p.ParseListDestructure(lexer.RightBracket, start)
	case lexer.LeftParenthesis:
		start := p.Advance().Position
		return p.ParseListDestructure(lexer.RightParenthesis, start)
	default:
		return nil
	}
}

func (p *Parser) parseSymbolDestruct() *ast.SymbolDestructure {
	tok := p.Advance()
	sym := &ast.SymbolDestructure{Identifier: tok.Source}
	return rangeFromToken(sym, tok)
}

func symbolDestructToSymbol(sym *ast.SymbolDestructure) *ast.Symbol {
	return &ast.Symbol{BaseNode: sym.BaseNode, Identifier: sym.Identifier}
}

func (p *Parser) ParseObjectDestructure() *ast.ObjectDestructure {
	var items []*ast.ObjectDestructureEntry
	parseSeries(p, &items, func() *ast.ObjectDestructureEntry {
		entry := &ast.ObjectDestructureEntry{}
		ident := p.parseSymbolDestruct()
		var end lexer.Position
		if curr := p.CurrentTokenKind(); curr == lexer.Colon {
			entry.Alias = ident
			p.Advance()
			// Parsing full destructure just for a better error
			value := p.ParseDestructure()
			if sym, ok := value.(*ast.SymbolDestructure); ok {
				entry.Object = symbolDestructToSymbol(sym)
			} else {
				p.Error(errors.Node(errors.ErrDestructPatAfterColon, value))
			}
			end = entry.Alias.GetRange().End
		} else if curr == lexer.Dot {
			entry.Object = symbolDestructToSymbol(ident)
			p.Advance()
			entry.Index = p.ParseDestructureInner()
			end = entry.Index.GetRange().End
		}
		curr := p.CurrentToken()
		if curr.Kind == lexer.ColonEqual {
			p.Error(errors.Token(errors.ErrColonEqual, curr))
			curr.Kind = lexer.Equal
		}
		if curr.Kind == lexer.Equal {
			if entry.Index != nil {
				p.Error(errors.Token(errors.ErrDestructInvalidEqual, curr))
			}
			p.Advance()
			entry.Default = p.ParseExpression(ExpressionBindingPower)
		}
		entry.SetPos(ident.Range.Start, end)
		return entry
	}, lexer.RightCurlyBrace, lexer.Comma, true)
	return &ast.ObjectDestructure{Values: items}
}

// Array or tuple
func (p *Parser) ParseListDestructure(end lexer.TokenType, start lexer.Position) ast.Destructure {
	var items []ast.Destructure
	parseSeries(p, &items, p.ParseDestructureInner, end, lexer.Comma, false)
	d := &ast.ListDestructure{
		Values: items,
		Tuple:  end == lexer.RightParenthesis,
	}
	return markStartEndPos(p, d, start)
}

func (p *Parser) ParseDestructureSeries() (vars []ast.Destructure) {
	for p.HasTokens() && p.CurrentTokenKind() != lexer.EndOfStatement {
		vars = append(vars, p.ParseDestructure())
		if p.CurrentTokenKind() != lexer.Comma {
			break
		}
		p.Expect(lexer.Comma)
	}
	return vars
}
