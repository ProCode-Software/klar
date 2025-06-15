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
		return ast.BadExpression{Token: kind}
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

func (p *Parser) ParseListType() ast.ListType {
	// Skip the [
	p.Advance()
	typ := p.ParseType(DefaultTypeBindingPower)
	p.Expect(lexer.RightBracket)
	return ast.ListType{Value: typ}
}

func (p *Parser) ParseTypeAlias() ast.Type {
	typ := p.Advance().Source
	if primType, is := ast.PrimitiveTypeMap[typ]; is {
		return ast.PrimitiveType{Primitive: primType}
	}
	return ast.TypeAlias{Identifier: typ}
}

func (p *Parser) ParseOptionalType(left ast.Type, bp BindingPower) ast.OptionalType {
	// Skip the ?
	p.Expect(lexer.Question)
	return ast.OptionalType{Value: left}
}

func (p *Parser) ParseUnionType(left ast.Type, bp BindingPower) (u ast.UnionType) {
	u.Options = make([]ast.Type, 1, 2)
	u.Options[0] = left
	for p.CurrentTokenKind() == lexer.Stroke {
		p.Advance()
		u.Options = append(u.Options, p.ParseType(bp))
	}
	return u
}

func (p *Parser) ParseGenericType(left ast.Type, bp BindingPower) ast.GenericType {
	params := make([]ast.Type, 0, 1)
	p.Expect(lexer.LessThan)
	if p.CurrentTokenKind() == lexer.GreaterThan {
		// At least 1 parameter required
		p.Error(errors.Token(errors.ErrExpectedParamInGeneric, p.CurrentToken()))
	}
	for p.WhileNotEndOr(lexer.GreaterThan) {
		params = append(params, p.ParseType(DefaultTypeBindingPower))
		if p.IsNotCurrentlyEndOr(lexer.GreaterThan) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.GreaterThan)
	return ast.GenericType{Name: left, Parameters: params}
}

func (p *Parser) ParseInterface(name string, inherited []ast.Type) ast.InterfaceDeclaration {
	var fields []ast.TypePair
	for p.WhileNotEndOr(lexer.RightCurlyBrace) {
		field := ast.TypePair{
			Key: p.expectMapIdent().Source,
			// If not ident, then it is unmatched
		}
		if p.CurrentTokenKind() == lexer.LeftParenthesis {
			// Parse function: #{ Kind() -> string }
			fn := ast.MethodType{}
			p.Advance() // (
			parseSeries(p, &fn.Parameters, func() ast.MethodTypeParam {
				var label, name string
				if p.Peek().Kind == lexer.Identifier {
					// Label: fib(length length: Int)
					label = p.expectNonNumericMapIdent().Source
					name = p.Expect(lexer.Identifier).Source
					p.Expect(lexer.Colon)
				} else if p.Peek().Kind == lexer.Colon {
					// Declared wih a name
					name = p.Expect(lexer.Identifier).Source
					p.Expect(lexer.Colon)
				}
				// Type
				return ast.MethodTypeParam{
					Type:       p.ParseType(CallBindingPower),
					Label:      label,
					Identifier: name,
				}
			}, lexer.RightParenthesis, lexer.Comma, false)
			if p.CurrentTokenKind() == lexer.Arrow {
				p.Advance()
				fn.ReturnType = p.ParseType(DefaultTypeBindingPower)
			}
			field.Value = fn
		} else {
			p.Expect(lexer.Colon)
			field.Value = p.ParseType(DefaultTypeBindingPower)
		}
		fields = append(fields, field)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.InterfaceDeclaration{
		Identifier:     name,
		InheritedTypes: inherited,
		Fields:         fields,
	}
}

func (p *Parser) ParseFunctionType(left ast.Type, bp BindingPower) ast.FunctionType {
	switch left.(type) {
	case ast.TypeAlias, ast.PrimitiveType:
		// Param must be in parentheses
		p.Error(errors.Node(errors.ErrParenRequiredFunc, left))
	case ast.TupleType:
	default:
		p.Error(errors.UnexpectedToken(p.CurrentToken()))
	}
	p.Expect(lexer.Arrow)
	return ast.FunctionType{
		Parameters: left.(ast.TupleType).Values,
		ReturnType: p.ParseType(DefaultTypeBindingPower),
	}
}

func (p *Parser) ParseTupleType() ast.TupleType {
	params := []ast.Type{}
	p.Advance() // (
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		params = append(params, p.ParseType(DefaultTypeBindingPower))
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	return ast.TupleType{Values: params}
}

func (p *Parser) ParseRestType(left ast.Type, bp BindingPower) ast.RestType {
	p.Advance()
	return ast.RestType{Value: left}
}
