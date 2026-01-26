package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseType(bp BindingPower) ast.Type {
	kind := p.CurrKind()
	left, handled := p.handleTypeNUD(kind)
	if !handled {
		p.unknownTokenErr()
		return &ast.BadExpression{Token: kind}
	}
	return p.ParseTypeLED(left, bp)
}

func (p *Parser) ParseTypeLED(left ast.Type, bp BindingPower) ast.Type {
	var handled bool
	for TypeBindingPowerMap[p.CurrKind()] > bp {
		kind := p.CurrKind()
		left, handled = p.handleTypeLED(
			kind, left, TypeBindingPowerMap[p.CurrKind()],
		)
		if !handled {
			p.unknownTokenErr()
			return &ast.BadExpression{Token: kind, Value: left}
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
	if left != nil {
		u.Options[0] = left
	}
	for p.CurrKind() == lexer.Stroke {
		p.Advance()
		u.Options = append(u.Options, p.ParseType(bp))
	}
	return u
}

func (p *Parser) ParseGenericType(left ast.Type, bp BindingPower) *ast.GenericType {
	params := make([]ast.Type, 0, 1)
	p.Expect(lexer.LessThan)
	if p.CurrKind() == lexer.GreaterThan {
		// At least 1 parameter required
		p.Error(errors.Token(errors.ErrEmptyGeneric, p.Curr()))
		params = nil
	}
	parseSeries(p, &params,
		func() ast.Type { return p.ParseType(DefaultTypeBindingPower) },
		lexer.GreaterThan, lexer.Comma, false,
	)
	return &ast.GenericType{Name: left, Parameters: params}
}

func (p *Parser) ParseFunctionType() *ast.FunctionType {
	p.Advance() // func
	if p.CurrKind() == lexer.Arrow {
		p.Advance()
		return &ast.FunctionType{ReturnType: p.ParseType(DefaultTypeBindingPower)}
	}
	fn := &ast.FunctionType{}
	if p.CurrKind() == lexer.LeftParenthesis {
		fn.Parameters = p.ParseTupleType()
	} else {
		p.Error(errors.Token(errors.ErrParenFuncTypeParams, p.Curr()))
		// Parse without parentheses
		fn.Parameters.Values = p.parseFuncTypeWithoutParen()
	}
	if p.CurrKind() == lexer.Arrow {
		p.Advance()
		fn.ReturnType = p.ParseType(DefaultTypeBindingPower)
	}
	return fn
}

func (p *Parser) parseFuncTypeWithoutParen() (values []*ast.TypePair) {
	for p.HasTokens() {
		pair := &ast.TypePair{}
		if p.PeekKind() == lexer.Colon {
			pair.Keys = append(pair.Keys, p.ParseIdentOrDiscard())
			p.Advance() // :
		}
		pair.Value = p.ParseType(DefaultTypeBindingPower)
		values = append(values, pair)
		if p.CurrKind() != lexer.Comma {
			break
		}
		p.Advance()
	}
	return
}

func (p *Parser) ParseTupleType() *ast.TupleType {
	p.Advance() // (
	var (
		tuple            = &ast.TupleType{}
		names            []ast.Identifier
		isType, hasColon bool
	)
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		// TODO: doesn't work if generic type inside (starts with ident)

		// (a, b, c) = 3 types
		// (a, b, c: Int) = 3 labels
		// (a, [b], c) = 3 types
		// (a, b: Int, c), ([a], b, c: Int) = invalid (mismatch)
		if k := p.CurrKind(); !isType &&
			isValidIdentOrDiscard(k) && p.PeekKind() != lexer.Dot { // Dot for member type
			names = append(names, p.ParseValidIdent())
			if p.CurrKind() == lexer.Colon {
				if isType {
					p.Error(errors.Token(errors.ErrMixTypeTupleLabels, p.Curr()))
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
		if len(tuple.Values) <= 1 && p.CurrKind() == lexer.RightParenthesis {
			// No comma
			break
		}
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	trailingComma := p.PeekBehind().Kind == lexer.Comma
	p.Expect(lexer.RightParenthesis)
	if len(names) > 0 {
		if hasColon {
			p.Error(errors.Token(errors.ErrMixTypeTupleLabels, p.Curr()))
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
	tuple.Single = len(tuple.Values) == 1 && trailingComma
	return tuple
}

func (p *Parser) ParseRestType() *ast.RestType {
	p.Advance() // ...
	return &ast.RestType{Value: p.ParseType(VariadicTypeBindingPower)}
}

func (p *Parser) ParseTypeNamespace(left *ast.TypeAlias, bp BindingPower) *ast.QualifiedTypeAlias {
	p.Advance() // .
	return &ast.QualifiedTypeAlias{
		Namespace: ast.Identifier{
			Name:     left.Identifier,
			Position: left.Range.Start,
			Len:      left.Range.LineLength(),
		},
		Identifier: p.ParseIdentifier(),
	}
}
