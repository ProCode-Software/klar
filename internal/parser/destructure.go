package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseDestructure() ast.Destructure {
	switch p.CurrentTokenKind() {
	case lexer.HashLeftCurlyBrace:
		p.Advance()
		return p.ParseObjectDestructure()
	case lexer.Identifier, lexer.LeftBracket, lexer.LeftParenthesis, lexer.Underscore:
		return p.ParseDestructureInner()
	}
	return nil
}

func (p *Parser) ParseDestructureInner() ast.Destructure {
	switch p.CurrentTokenKind() {
	case lexer.Identifier:
		return p.parseSymbolDestruct()
	case lexer.Underscore:
		return rangeFromToken(&ast.Discard{}, p.Advance())
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
		kind := p.CurrentTokenKind()
		p.unknownTokenErr()
		return &ast.BadExpression{Token: kind}
	}
}

func (p *Parser) parseSymbolDestruct() *ast.SymbolDestructure {
	tok := p.Advance()
	name := tok.Source
	sym := &ast.SymbolDestructure{
		Identifier: name,
		Constant:   isConstant(name),
	}
	return rangeFromToken(sym, tok)
}

func symbolDestructToSymbol(sym *ast.SymbolDestructure) *ast.Symbol {
	return &ast.Symbol{BaseNode: sym.BaseNode, Identifier: sym.Identifier}
}

func (p *Parser) ParseObjectDestructure() *ast.ObjectDestructure {
	var items []*ast.ObjectDestructureEntry
	if p.errorIfEmptyDestruct(lexer.RightCurlyBrace) {
		return &ast.ObjectDestructure{}
	}
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
		p.Expect(lexer.Comma, lexer.EndOfStatement)
		return entry
	}, lexer.RightCurlyBrace, 0, true)
	return &ast.ObjectDestructure{Values: items}
}

func (p *Parser) errorIfEmptyDestruct(endKind lexer.TokenType) bool {
	if p.CurrentTokenKind() == endKind {
		start := p.Tokens[p.Index-1].Position
		end := ranges.FromToken(p.CurrentToken()).End
		p.Error(errors.Range(errors.ErrEmptyDestructure, ranges.BetweenPos(start, end)))
		p.Advance()
		return true
	}
	return false
}

// Array or tuple
func (p *Parser) ParseListDestructure(end lexer.TokenType, start lexer.Position) ast.Destructure {
	if p.errorIfEmptyDestruct(end) {
		return &ast.BadExpression{Token: end}
	}
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

func (p *Parser) ParseAssignLeft() (vars []ast.Assignable) {
	for p.HasTokens() && p.CurrentTokenKind() != lexer.EndOfStatement {
		dest := p.ParseDestructure()
		curr := p.CurrentTokenKind()
		if dest, ok := dest.(*ast.SymbolDestructure); ok &&
			!isAssignment(curr) && curr != lexer.Comma && curr != lexer.Colon {
			res, handled := p.handleLED(
				curr, symbolDestructToSymbol(dest), ExpressionBindingPower,
			)
			if !handled {
				p.unknownTokenErr()
				res = dest
			}
			if !p.validateAssignable(res) {
				res = &ast.BadExpression{Value: res}
			}
			vars = append(vars, res.(ast.Assignable))
		} else {
			vars = append(vars, dest)
		}
		if p.CurrentTokenKind() != lexer.Comma {
			break
		}
		p.Expect(lexer.Comma)
	}
	return vars
}

// Validate the += or -= operator at type-check time
func (p *Parser) ParseDestructureVars() *ast.DestructureVars {
	return &ast.DestructureVars{Values: p.ParseAssignLeft()}
}
