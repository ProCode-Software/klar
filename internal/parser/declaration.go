package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

/*
Type declaration:

	type Type = Int -- type alias
	type Type { ... } -- struct or enum
	type Type {} -- if empty, then it is a struct
	type #Type { ... } -- interface
	type #Type -- type tag (interface with no requirements, manually implemented)
*/
func (p *Parser) ParseTypeDeclaration() ast.TypeDeclaration {
	p.Expect(lexer.Type)
	isIntf := p.CurrentTokenKind() == lexer.Hash
	if isIntf {
		p.Advance()
	}
	var (
		name      = p.Expect(lexer.Identifier)
		inherited []ast.Type
	)

	switch p.CurrentTokenKind() {
	case lexer.Equal:
		// Type alias
		p.Advance()
		return ast.TypeAliasDeclaration{
			Identifier: name.Source,
			Type:       p.ParseType(DefaultTypeBindingPower),
		}
	case lexer.Colon:
		// Inherited struct
		p.Advance()
		for p.WhileNotEndOr(lexer.LeftCurlyBrace) {
			// Alias or generic
			inherited = append(inherited, p.ParseType(UnionTypeBindingPower))
			if p.CurrentTokenKind() != lexer.LeftCurlyBrace {
				p.Expect(lexer.Comma)
			}
		}
		fallthrough
	case lexer.LeftCurlyBrace:
		// Struct or enum
		p.Expect(lexer.LeftCurlyBrace)
		if isIntf {
			return p.ParseInterface(name.Source, inherited)
		}
		if p.CurrentTokenKind() == lexer.RightCurlyBrace {
			// Empty struct
			p.Advance()
			return ast.StructDeclaration{
				Identifier:     name.Source,
				InheritedTypes: inherited,
			}
		}

		var isEnum bool
		// Leading | for formatting
		// type Color {
		// 	| Red
		// 	| Blue
		// }
		if p.CurrentTokenKind() == lexer.Stroke {
			isEnum = true
			p.Advance()
		}
		fieldName := p.expectNonNumericMapIdent()
		// Struct fields always need a type
		// 	range: Int = 1000
		//	range = 1000 // Incorrect
		if isEnum || p.IsCurrently(lexer.Equal, lexer.Stroke, lexer.LeftParenthesis) {
			// Can't use reserved keyword as enum member
			if fieldName.Kind != lexer.Identifier {
				p.Error(errors.Token(errors.ErrReservedKeyword, fieldName))
			}
			return p.ParseEnum(name.Source, fieldName, inherited)
		} else if p.CurrentTokenKind() == lexer.Colon {
			return p.ParseStruct(name.Source, fieldName, inherited)
		} else {
			p.Error(errors.Token(errors.ErrCannotTellStructOrEnum, fieldName))
		}
	case lexer.EndOfStatement:
		// Type tag if interface
		if isIntf {
			return ast.InterfaceDeclaration{
				Tag:            true,
				Identifier:     name.Source,
				InheritedTypes: inherited,
			}
		}
		fallthrough
	default:
		// Some other token or unassigned type (if EOS)
		p.Error(errors.Token(errors.ErrExpectedTypeAssignment, p.CurrentToken()))
		p.Advance()
		return ast.TypeAliasDeclaration{Identifier: name.Source}
	}
	panic("unhandled value in type declaration")
}

func (p *Parser) ParseEnum(
	typeName string, firstItem lexer.Token, inherited []ast.Type,
) ast.EnumDeclaration {
	var (
		isFirst = true
		items   []ast.EnumItem
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		var item ast.EnumItem
		if isFirst {
			item = rangeFromToken(ast.EnumItem{Identifier: firstItem.Source}, firstItem)
			isFirst = false
		} else {
			tok := p.Expect(lexer.Identifier)
			item = rangeFromToken(ast.EnumItem{Identifier: tok.Source}, tok)
		}
		if p.CurrentTokenKind() == lexer.LeftParenthesis {
			p.Advance()
			// Don't allow lambdas as parameter types
			parseSeries(p, &item.Parameters,
				func() ast.Type { return p.ParseType(FunctionTypeBindingPower) },
				lexer.RightParenthesis, lexer.Comma, false,
			)
		}
		if p.CurrentTokenKind() == lexer.Equal {
			if item.Parameters != nil {
				p.Error(errors.Token(errors.ErrEnumParamAndValue, p.CurrentToken()))
			}
			p.Advance()
			item.Value = p.ParseExpression(PrimaryBindingPower)
		}
		item = markEndPos(p, item)
		items = append(items, item)
		if p.CurrentTokenKind() == lexer.EndOfStatement {
			p.Advance()
			break
		}
		if p.IsNotCurrentlyEndOr(lexer.RightCurlyBrace) {
			p.Expect(lexer.Stroke)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.EnumDeclaration{
		Identifier: typeName,
		Values:     items,
		Inherited:  inherited,
	}
}

func (p *Parser) ParseStruct(
	typeName string, firstField lexer.Token, inherited []ast.Type,
) ast.StructDeclaration {
	var (
		isFirst = true
		fields  []ast.StructField
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		var field ast.StructField
		if isFirst {
			// First is currently at colon
			field = ast.StructField{Identifier: firstField.Source}
			field = rangeFromToken(field, firstField)
			isFirst = false
		} else {
			if !p.isMapIdentifier() {
				p.Error(errors.ExpectedToken(lexer.Identifier, p.CurrentToken()))
			}
			field = ast.StructField{Identifier: p.Advance().Source}
		}
		// Type
		p.ExpectError(errors.Token(
			errors.ErrRequiredStructFieldType, p.CurrentToken(),
		), lexer.Colon)
		field.Type = p.ParseType(DefaultTypeBindingPower)
		// Default value
		if p.CurrentTokenKind() == lexer.Equal {
			p.Advance()
			field.Value = p.ParseExpression(DefaultBindingPower)
		}
		field = markEndPos(p, field)
		fields = append(fields, field)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.StructDeclaration{
		Identifier:     typeName,
		InheritedTypes: inherited,
		Fields:         fields,
	}
}

func (p *Parser) ParseFuncDeclaration() ast.FunctionDeclaration {
	p.Expect(lexer.Func)
	f := ast.FunctionDeclaration{}
	rec := p.Expect(lexer.Identifier)

	// Struct receiver
	// 	func Person.greet()
	if p.CurrentTokenKind() == lexer.Dot {
		p.Advance()
		alias := ast.TypeAlias{Identifier: rec.Source}
		f.Struct = rangeFromToken(alias, rec)
		f.Identifier = p.Expect(lexer.Identifier).Source
	} else {
		f.Identifier = rec.Source
	}
	// Generic:
	//	func get<T, U>(a: T, b: [U]) -> T
	// Can't be assigned, only inferred
	if p.CurrentTokenKind() == lexer.LessThan {
		generics := make([]ast.Symbol, 0, 1)
		p.Advance()
		for p.WhileNot(lexer.GreaterThan) {
			tok := p.Expect(lexer.Identifier)
			item := ast.Symbol{Identifier: tok.Source}
			item = rangeFromToken(item, tok)
			generics = append(generics, item)
			if p.CurrentTokenKind() != lexer.GreaterThan {
				p.Expect(lexer.Comma)
			}
		}
		gt := p.Expect(lexer.GreaterThan)
		if len(generics) == 0 {
			p.Error(errors.Token(errors.ErrEmptyGeneric, gt))
			generics = nil
		}
		f.GenericParams = generics
	}
	// Params
	p.Expect(lexer.LeftParenthesis)
	var isTrailingType bool
	applyType := func(t ast.Type, v ast.Expression) {
		isTrailingType = false
		for i := len(f.Parameters) - 1; i >= 0; i-- {
			if f.Parameters[i].Type != nil {
				break
			}
			f.Parameters[i].Type = t
			f.Parameters[i].Default = v
		}
	}
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		param := ast.FunctionParam{}
		param.BaseNode.Range.Start = p.CurrentToken().Position

		if peek := p.Peek().Kind; peek == lexer.Identifier ||
			peek == lexer.EndOfStatement {
			// Optional label:
			// 	func replace(src, with replacement: String)
			param.Label = p.expectNonNumericMapIdent().Source
			if peek == lexer.EndOfStatement {
				p.Advance()
			}
		}
		// Normal identifier
		param.Identifier = p.Expect(lexer.Identifier).Source
		// Parse type: still allow trailing type (example above)
		if p.CurrentTokenKind() == lexer.Colon {
			p.Advance()
			param.Type = p.ParseType(DefaultTypeBindingPower)
			applyType(param.Type, nil)
		} else {
			isTrailingType = true
		}
		// Default value:
		// 	func List.join(by by: String = ", ")
		if p.CurrentTokenKind() == lexer.Equal {
			p.Advance()
			param.Default = p.ParseExpression(DefaultBindingPower)
			applyType(param.Type, param.Default)
		}
		param.BaseNode.Range.End = p.lastTokEnd()
		f.Parameters = append(f.Parameters, param)
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	if isTrailingType {
		// Last parameters need a type
		p.Error(errors.Range(
			errors.ErrMissingFuncParamType,
			f.Parameters[len(f.Parameters)-1].Range,
		))
	}

	// Return type: the arrow. Can be inferred
	if p.CurrentTokenKind() == lexer.Arrow {
		p.Advance()
		f.ReturnType = p.ParseType(DefaultTypeBindingPower)
	}

	// Body: Externally implemented functions may not have a body
	//	@external(js: "./date.js", name: "now")
	// 	func Date.now() -> Date
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		f.Body = p.ParseBlock()
	} else if p.CurrentTokenKind() == lexer.Equal {
		p.Advance()
		f.Expression = p.ParseExpression(ExpressionBindingPower)
	}
	return f
}

func (p *Parser) ParseAttribute() (d ast.Attribute) {
	p.Expect(lexer.At)
	p.isAttribute = true
	d.Decorator = p.Expect(lexer.Identifier).Source
	if p.CurrentTokenKind() == lexer.LeftParenthesis {
		call := p.ParseCallExpression(nil, CallBindingPower)
		d.Args = call.Args
	}
	p.isAttribute = false
	return d
}

func (p *Parser) ParsePublicModifier() ast.Statement {
	p.Expect(lexer.Public)
	stmt := p.ParseStatement()
	if pub, ok := stmt.(ast.Publicizable); ok {
		stmt = pub.Publicize() // Set Public to true
	} else {
		p.Error(errors.Node(errors.ErrInvalidPublic, stmt))
	}
	return stmt
}
