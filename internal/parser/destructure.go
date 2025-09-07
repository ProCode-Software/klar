package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/pkg/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseDestructure() ast.Destructure {
	if p.CurrKind() == lexer.HashLeftCurlyBrace {
		p.Advance()
		return p.ParseObjectDestructure()
	}
	return p.ParseDestructureInner()
}

func (p *Parser) ParseDestructureInner() ast.Destructure {
	switch kind := p.CurrKind(); kind {
	case lexer.Identifier:
		return p.ParseValidIdent().Symbol()
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
	suggest := name + "." + string(printer.PrintTokens(tokens))
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
		ident := p.ParseMapIdentifier(true)
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
		if p.isEqualOrColonEqualAndError() {
			p.Advance()
			entry.Default = p.ParseExpression(ExpressionBindingPower)
		}
		entry.SetPos(ident.Position, end)
		p.Expect(lexer.Comma, lexer.EndOfStatement)
		return entry
	}, lexer.RightCurlyBrace, 0, true)
	return &ast.ObjectDestructure{Values: items}
}

func (p *Parser) errorIfEmptyDestruct(endKind lexer.TokenType) bool {
	if p.CurrKind() == endKind {
		start := p.Tokens[p.Index-1].Position
		end := ranges.FromToken(p.Curr()).End
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
	parseSeries(p, &vars, p.ParseDestructure, 0, lexer.Comma, false)
	return vars
}

func (p *Parser) ParseAssignLeft() (vars []ast.Assignable) {
	for p.HasTokens() && p.CurrKind() != lexer.EndOfStatement {
		dest := p.ParseDestructure()
		curr := p.CurrKind()
		if dest2, ok := dest.(*ast.Symbol); ok &&
			!isAssignment(curr) && curr != lexer.Comma && curr != lexer.Colon {
			res, handled := p.handleLED(curr, dest2, ExpressionBindingPower)
			if !handled {
				p.unknownTokenErr()
				res = dest2
			}
			if !p.validateAssignable(res) {
				res = &ast.BadExpression{Value: res}
			}
			vars = append(vars, res.(ast.Assignable))
		} else {
			vars = append(vars, dest)
		}
		if p.CurrKind() != lexer.Comma {
			break
		}
		p.Advance()
	}
	return vars
}

// Validate the += or -= operator at type-check time
func (p *Parser) ParseDestructureVars() *ast.DestructureVars {
	return &ast.DestructureVars{Values: p.ParseAssignLeft()}
}

// For lambda params or 'for' loop variables. Parentheses are not parsed.
func (p *Parser) ParseDestructureTypePairs(withDefaults bool) (pairs []*ast.DestructureTypePair) {
	parseSeries(p, &pairs, func() *ast.DestructureTypePair {
		pair := &ast.DestructureTypePair{}
		parseSeries(p, &pair.Keys, p.ParseDestructure, 0, lexer.Comma, false)
		if p.CurrKind() == lexer.Colon {
			p.Advance()
			pair.Type = p.ParseType(DefaultTypeBindingPower)
		}
		if withDefaults && p.isEqualOrColonEqualAndError() {
			p.Advance()
			pair.Value = p.ParseExpression(ExpressionBindingPower)
		}
		return pair
	}, 0, lexer.Comma, false)
	return pairs
}
