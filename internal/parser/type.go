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
	var params []ast.Type
	switch left := left.(type) {
	case *ast.TupleType:
		params = left.Values
	case *ast.FunctionType:
		// Allow (Int) -> (Int) -> Int
		params = []ast.Type{left}
	default:
		p.Error(errors.Node(errors.ErrParenRequiredFunc, left))
	}
	p.Expect(lexer.Arrow)
	return &ast.FunctionType{
		Parameters: params,
		ReturnType: p.ParseType(DefaultTypeBindingPower),
	}
}

func (p *Parser) ParseTupleType() *ast.TupleType {
	var params []ast.Type
	p.Advance() // (
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		params = append(params, p.ParseType(DefaultTypeBindingPower))
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	return &ast.TupleType{Values: params}
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
