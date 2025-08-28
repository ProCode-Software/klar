package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseType(bp BindingPower) ast.Type {
	kind := p.CurrentTokenKind()
	left, handled := p.handleTypeNUD(kind)
	if !handled {
		p.unknownTokenErr()
		return &ast.BadExpression{Token: kind}
	}
	for TypeBindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleTypeLED(
			kind, left, TypeBindingPowerMap[p.CurrentTokenKind()],
		)
		if !handled {
			p.unknownTokenErr()
			continue
		}
	}
	return left
}

func (p *Parser) ParseListType() *ast.ListType {
	// Skip the [
	p.Advance()
	typ := p.ParseType(DefaultTypeBindingPower)
	p.Expect(lexer.RightBracket)
	return &ast.ListType{Value: typ}
}

func (p *Parser) ParseTypeAlias() ast.Type {
	typ := p.Advance().Source
	if primType, is := ast.PrimitiveTypeMap[typ]; is {
		return &ast.PrimitiveType{Primitive: primType}
	}
	return &ast.TypeAlias{Identifier: typ}
}

func (p *Parser) ParseOptionalType(left ast.Type, bp BindingPower) *ast.OptionalType {
	// Skip the ?
	p.Expect(lexer.Question)
	return &ast.OptionalType{Value: left}
}

func (p *Parser) ParseUnionType(left ast.Type, bp BindingPower) *ast.UnionType {
	u := &ast.UnionType{}
	u.Options = make([]ast.Type, 1, 2)
	u.Options[0] = left
	for p.CurrentTokenKind() == lexer.Stroke {
		p.Advance()
		u.Options = append(u.Options, p.ParseType(bp))
	}
	return u
}

func (p *Parser) ParseGenericType(left ast.Type, bp BindingPower) *ast.GenericType {
	params := make([]ast.Type, 0, 1)
	p.Expect(lexer.LessThan)
	if p.CurrentTokenKind() == lexer.GreaterThan {
		// At least 1 parameter required
		p.Error(errors.Token(errors.ErrEmptyGeneric, p.CurrentToken()))
		params = nil
	}
	for p.WhileNotEndOr(lexer.GreaterThan) {
		params = append(params, p.ParseType(DefaultTypeBindingPower))
		if p.IsNotCurrentlyEndOr(lexer.GreaterThan) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.GreaterThan)
	return &ast.GenericType{Name: left, Parameters: params}
}

func (p *Parser) ParseFunctionType(left ast.Type, bp BindingPower) *ast.FunctionType {
	var tuple *ast.TupleType
	switch left := left.(type) {
	case *ast.TupleType:
		tuple = left
	case *ast.FunctionType:
		// Allow (Int) -> (Int) -> Int
		tuple = left.Parameters
	default:
		p.Error(errors.Node(errors.ErrParenRequiredFunc, left))
	}
	p.Expect(lexer.Arrow)
	return &ast.FunctionType{
		Parameters: tuple,
		ReturnType: p.ParseType(DefaultTypeBindingPower),
	}
}

func (p *Parser) ParseTupleType() *ast.TupleType {
	p.Advance() // (
	var (
		tuple            = &ast.TupleType{}
		names            []ast.Identifier
		isType, hasColon bool
	)
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		// (a, b, c) = 3 types
		// (a, b, c: Int) = 3 labels
		// (a, [b], c) = 3 types
		// (a, b: Int, c), ([a], b, c: Int) = invalid (mismatch)
		if k := p.CurrentTokenKind(); !isType &&
			isValidIdentifier(k) && p.Peek().Kind != lexer.Dot {
			names = append(names, p.ParseIdentifier())
			if p.CurrentTokenKind() == lexer.Colon {
				if isType {
					p.Error(errors.Token(errors.ErrMixTypeTupleLabels, p.CurrentToken()))
				}
				p.Advance() // :
				pair := &ast.TypePair{
					Keys: names, Value: p.ParseType(DefaultTypeBindingPower),
				}
				tuple.Values = append(tuple.Values, pair)
				names = names[:0]
				hasColon = true
			}
		} else {
			isType = true
			t := p.ParseType(DefaultTypeBindingPower)
			tuple.Values = append(tuple.Values, &ast.TypePair{Value: t})
			if hasColon {
				p.Error(errors.Node(errors.ErrMixTypeTupleLabels, t))
			}
		}
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	if len(names) > 0 {
		if hasColon {
			p.Error(errors.Token(errors.ErrMixTypeTupleLabels, p.CurrentToken()))
		}
		for _, name := range names {
			tuple.Values = append(tuple.Values, &ast.TypePair{
				Value: &ast.TypeAlias{
					BaseNode:   name.BaseNode(),
					Identifier: name.Name,
				},
			})
		}
	}
	return tuple
}

func (p *Parser) ParseRestType() *ast.RestType {
	p.Advance() // ...
	return &ast.RestType{Value: p.ParseType(VariadicTypeBindingPower)}
}

func (p *Parser) ParseTypeNamespace(left *ast.TypeAlias, bp BindingPower) *ast.TypeAlias {
	p.Advance() // .
	left.Namespace = p.Expect(lexer.Identifier).Source
	return left
}
