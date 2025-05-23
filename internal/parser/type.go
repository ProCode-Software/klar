package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
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
	p.Expect(lexer.Question)
	return ast.OptionalType{Value: left}
}

// Either + or |
func (p *Parser) ParseUnionType(left ast.Type, bp BindingPower) ast.UnionType {
	op := p.Advance().Kind
	right := p.ParseType(bp, op == lexer.Stroke)
	return ast.UnionType{
		Left:     left,
		Right:    right,
		Operator: op,
	}
}

func (p *Parser) ParseGenericType(left ast.Type, bp BindingPower) ast.GenericType {
	return ast.GenericType{}
}

func (p *Parser) ParseInterfaceType() ast.InterfaceType {
	fields := []ast.TypePair{}
	p.Advance() // #{
	for p.IsNot(lexer.RightCurlyBrace) {
		field := ast.TypePair{
			Key: p.Expect(lexer.Identifier).Source,
		}
		if p.CurrentTokenKind() == lexer.LeftParenthesis {
			// Parse function: #{ Kind() -> string }
			fn := ast.FunctionType{}
			p.Advance()
			for p.IsNot(lexer.RightParenthesis) {
				fn.Parameters = append(fn.Parameters, p.ParseType(CallBindingPower, true))
				if p.CurrentTokenKind() != lexer.RightParenthesis {
					p.Expect(lexer.Comma)
				}
			}
			p.Expect(lexer.RightParenthesis)
			if p.CurrentTokenKind() == lexer.Arrow {
				p.Advance()
				fn.ReturnType = p.ParseType(DefaultTypeBindingPower, true)
			}
			field.Value = fn
		} else {
			p.Expect(lexer.Colon)
			field.Value = p.ParseType(AssignBindingPower, false)
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
	if _, ok := left.(ast.TupleType); !ok {
		panic(errors.UnknownTokenError(p.CurrentToken()))
	}
	p.Expect(lexer.Arrow)
	return ast.FunctionType{
		Parameters: left.(ast.TupleType).Values,
		ReturnType: p.ParseType(DefaultTypeBindingPower, false),
	}
}
func (p *Parser) ParseTupleType() ast.TupleType {
	params := []ast.Type{}
	p.Advance() // (
	for p.IsNot(lexer.RightParenthesis) {
		params = append(params, p.ParseType(DefaultTypeBindingPower, true))
		if p.CurrentTokenKind() != lexer.RightParenthesis {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	return ast.TupleType{Values: params}
}
