package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseType(bp BindingPower, simple bool) ast.Type {
	kind := p.CurrentTokenKind()
	left, handled := p.handleTypeNUD(kind)
	if !handled {
		noHandlerError(p, "TypeNUD")
	}
	for BindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleTypeLED(kind, left, bp)
		if !handled {
			noHandlerError(p, "TypeLED")
		}
	}
	return left
}

func (p *Parser) ParseListType() ast.ListType {
	// Skip the [
	p.Advance()
	typ := p.ParseType(DefaultBindingPower, true)
	p.Expect(lexer.RightBracket)
	return ast.ListType{Value: typ}
}

func (p *Parser) ParseTypeAlias() ast.SimpleType {
	typ := p.Advance().Source
	if primType, is := ast.PrimitiveTypeMap[typ]; is {
		return ast.PrimitiveType{Primitive: primType}
	}
	return ast.TypeAlias{Identifier: typ}
}

func (p *Parser) ParseOptionalType(left ast.Type, bp BindingPower) ast.OptionalType {
	// Skip the ?
	p.Advance()
	return ast.OptionalType{Value: left}
}

// Either + or |
func (p *Parser) ParseUnionType(left ast.Type, bp BindingPower) ast.UnionType {
	op := p.Advance().Kind
	right := p.ParseType(bp, op == lexer.Alternative)
	return ast.UnionType{
		Left: left,
		Right: right,
		Operator: op,
	}
}

func (p *Parser) ParseGenericType(left ast.Type, bp BindingPower) ast.GenericType {
	return ast.GenericType{}
}