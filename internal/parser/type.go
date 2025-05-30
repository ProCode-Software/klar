package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseType(bp BindingPower) ast.SimpleType {
	kind := p.CurrentTokenKind()
	left, handled := p.handleTypeNUD(kind)
	if !handled {
		p.unknownTokenErr(false)
		return ast.BadExpression{}
	}
	for TypeBindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleTypeLED(
			kind, left, TypeBindingPowerMap[p.CurrentTokenKind()],
		)
		if !handled {
			p.unknownTokenErr(true)
			continue
		}
	}
	return left.(ast.SimpleType)
}

func (p *Parser) ParseComplexType(bp BindingPower) ast.Type {
	if p.CurrentTokenKind() == lexer.HashLeftCurlyBrace {
		return p.ParseInterfaceType()
	}
	return p.ParseType(bp)
}

func (p *Parser) ParseListType() ast.ListType {
	// Skip the [
	p.Advance()
	typ := p.ParseType(DefaultTypeBindingPower)
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
	params := make([]ast.SimpleType, 0, 1)
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

func (p *Parser) ParseInterfaceType() ast.InterfaceType {
	fields := []ast.TypePair{}
	p.Advance() // #{
	for p.WhileNotEndOr(lexer.RightCurlyBrace) {
		if !p.isMapIdentifier() {
			errors.ExpectedToken(lexer.Identifier, p.CurrentToken())
		}
		field := ast.TypePair{
			Key: p.Advance().Source,
			// If not ident, then it is unmatched
		}
		if p.CurrentTokenKind() == lexer.LeftParenthesis {
			// Parse function: #{ Kind() -> string }
			fn := ast.FunctionType{}
			p.Advance()
			for p.WhileNot(lexer.RightParenthesis) {
				fn.Parameters = append(fn.Parameters, p.ParseType(CallBindingPower))
				if p.CurrentTokenKind() != lexer.RightParenthesis {
					p.Expect(lexer.Comma)
				}
			}
			p.Expect(lexer.RightParenthesis)
			if p.CurrentTokenKind() == lexer.Arrow {
				p.Advance()
				fn.ReturnType = p.ParseType(DefaultTypeBindingPower)
			}
			field.Value = fn
		} else {
			p.Expect(lexer.Colon)
			field.Value = p.ParseType(AssignBindingPower)
		}
		fields = append(fields, field)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.InterfaceType{
		Fields: fields,
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
