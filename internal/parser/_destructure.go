package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseDestructure() ast.Destructure {
	if p.CurrKind() == lexer.HashLeftCurlyBrace {
		p.Advance()
		return p.ParseObjectDestructure()
	}
	d := p.ParseDestructureInner()
	// Continue parsing invalid expressions starting with identifiers
	if expr, ok := d.(ast.Expression); ok && p.CurrKind() != lexer.In {
		if expr, ok = p.TryParseLED(expr, ExpressionBindingPower); ok {
			p.Error(errors.Node(errors.ErrInvalidAssignment, expr))
			return &ast.BadExpression{Value: expr}
		}
	}
	return d
}

func (p *Parser) ParseDestructureInner() ast.Destructure {
	switch kind := p.CurrKind(); kind {
	case lexer.Identifier:
		id := p.ParseValidIdent().Symbol()
		if p.CurrKind() == lexer.Ellipsis {
			p.Advance()
			rest := &ast.RestExpression{Expression: id}
			return markStartEndPos(p, rest, id.Range.Start)
		}
		return id
	case lexer.Underscore:
		us := rangeFromToken(&ast.Discard{}, p.Advance())
		if p.CurrKind() == lexer.Ellipsis {
			p.Advance()
			rest := &ast.RestExpression{Expression: us}
			markStartEndPos(p, rest, us.Range.Start)
			// No '_' with '...'
			p.Error(errors.Node(errors.ErrUnderscoreWithRest, rest))
			return rest
		}
		return us
	case lexer.LeftCurlyBrace:
		start := p.Advance().Position
		return markStartEndPos(p, p.ParseObjectDestructure(), start)
	case lexer.LeftBracket:
		start := p.Advance().Position
		return p.ParseListDestructure(lexer.RightBracket, start)
	case lexer.LeftParenthesis:
		start := p.Advance().Position
		return p.ParseListDestructure(lexer.RightParenthesis, start)
	case lexer.Ellipsis:
		return rangeFromToken(&ast.RestExpression{}, p.Advance())
	default:
		if isValidIdentifier(kind) {
			return p.ParseValidIdent().Symbol()
		}
		parsed := p.ParseExpression(ExpressionBindingPower)
		p.Error(errors.Node(errors.ErrInvalidAssignment, parsed))
		// p.unknownTokenErr() // p.Error(errors.UnexpectedToken(p.AdvanceNonBoundary()))
		return &ast.BadExpression{Token: kind, Value: parsed}
	}
}

func makeColonDestructHint(tokens []lexer.Token, name string, node ast.Destructure) string {
	var kind string
	switch n := node.(type) {
	case *ast.ListDestructure:
		if n.Tuple {
			kind = "tuple"
		} else {
			kind = "list"
		}
	case *ast.ObjectDestructure:
		kind = "object"
	}
	suggest := name + "." + string(printTokens(tokens))
	return fmt.Sprintf("Did you mean %s for %s destructuring?", errors.Quote(suggest), kind)
}

func (p *Parser) ParseObjectDestructure() *ast.ObjectDestructure {
	var items []*ast.ObjectDestructureEntry
	if p.errorIfEmptyDestruct(lexer.RightCurlyBrace) {
		return &ast.ObjectDestructure{}
	}
	parseSeries(p, &items, func() *ast.ObjectDestructureEntry {
		entry := &ast.ObjectDestructureEntry{}
		identTok := p.Curr()
		ident := p.ParseMapIdentifier(includingNumber)
		var end lexer.Position
		if curr := p.CurrKind(); curr == lexer.Colon {
			entry.Alias = ident
			p.Advance()
			// Parsing full destructure just for a better error
			destructStart := p.Index
			value := p.ParseDestructure()
			if sym, ok := value.(*ast.Symbol); ok {
				entry.Object = sym
			} else {
				err := errors.Node(errors.ErrDestructPatAfterColon, value)
				err.Hint(makeColonDestructHint(
					p.Tokens[destructStart:p.Index], ident.Name, value,
				))
				p.Error(err)
			}
			end = entry.Alias.GetRange().End
		} else if curr == lexer.Dot {
			entry.Object = ident.Symbol()
			p.AdvanceNonBoundary()
			entry.Index = p.ParseDestructureInner()
			end = entry.Index.GetRange().End
		} else {
			p.validateIdentifier(identTok)
			entry.Object = ident.Symbol()
		}
		if p.isEqual() {
			p.Advance()
			entry.Default = p.ParseExpression(ExpressionBindingPower)
		}
		entry.SetPos(ident.Position, end)
		p.Expect(lexer.Comma, lexer.Newline)
		return entry
	}, lexer.RightCurlyBrace, 0, true)
	return &ast.ObjectDestructure{Values: items}
}

func (p *Parser) errorIfEmptyDestruct(endKind lexer.TokenType) bool {
	if p.CurrKind() == endKind {
		start := p.Tokens[p.Index-1].Position
		end := p.Curr().End()
		p.Error(errors.Range(errors.ErrEmptyDestructure, ranges.FromPosition(start, end)))
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
	parseSeries(p, &items, func() ast.Destructure {
		dest := p.ParseDestructureInner()
		if c := p.CurrKind(); c == lexer.Equal || c == lexer.ColonEqual {
			p.Error(errors.Token(errors.ErrDestructInvalidEqual, p.Curr()))
			// Just parse it
			p.Advance()
			p.ParseExpression(DefaultBindingPower)
		}
		return dest
	}, end, lexer.Comma, false)
	d := &ast.ListDestructure{
		Values: items,
		Tuple:  end == lexer.RightParenthesis,
	}
	return markStartEndPos(p, d, start)
}

func (p *Parser) ParseDestructureSeries() (vars []ast.Expression) {
	parseSeries(p, &vars, func() ast.Expression { return p.ParseDestructure() }, 0, lexer.Comma, false)
	return vars
}
