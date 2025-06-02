package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseTypeDeclaration() ast.TypeDeclaration {
	p.Expect(lexer.Type)
	var (
		name      = p.Expect(lexer.Identifier)
		inherited []ast.Type
	)

	switch p.CurrentTokenKind() {
	case lexer.Equal:
		// Type
		p.Advance()
		return ast.TypeAliasDeclaration{
			Identifier: name.Source,
			Type:       p.ParseComplexType(DefaultTypeBindingPower),
		}
	case lexer.Colon:
		// Inherited struct
		p.Advance()
		for p.WhileNotEndOr(lexer.LeftCurlyBrace) {
			inherited = append(inherited, p.ParseTypeAlias())
			if p.CurrentTokenKind() != lexer.LeftCurlyBrace {
				p.Expect(lexer.Comma)
			}
		}
		fallthrough
	case lexer.LeftCurlyBrace:
		// Struct or enum
		p.Expect(lexer.LeftCurlyBrace)

		// Leading | for formatting
		// type Color {
		// 	| Red
		// 	| Blue
		// }
		if p.CurrentTokenKind() == lexer.Stroke {
			p.Advance()
		}
		if !p.isMapIdentifier() {
			errors.ExpectedToken(lexer.Identifier, p.CurrentToken())
		}
		fieldName := p.Advance()
		// Struct fields always need a type
		// 	range: Int = 1000
		//	range = 1000 // Incorrect
		if p.CurrentTokenKind() == lexer.Colon {
			return p.ParseStruct(name.Source, fieldName.Source, inherited)
		} else if p.IsCurrently(lexer.Equal, lexer.Stroke) {
			// Can't use reserved keyword as enum member
			if fieldName.Kind != lexer.Identifier {
				p.Error(errors.Token(errors.ErrReservedKeyword, fieldName))
			}
			return p.ParseEnum(name.Source, fieldName.Source)
		}
	default:
		// Some other token or unassigned type (if EOS)
		p.Error(errors.Token(errors.ErrExpectedTypeAssignment, p.CurrentToken()))
		p.Advance()
		return ast.TypeAliasDeclaration{Identifier: name.Source}
	}
	return nil
}

func (p *Parser) ParseEnum(typeName, firstItem string) ast.EnumDeclaration {
	var (
		isFirst = true
		items   []ast.EnumItem
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		var item ast.EnumItem
		if isFirst {
			item = ast.EnumItem{Identifier: firstItem}
			isFirst = false
			if p.CurrentTokenKind() == lexer.Stroke {
				items = append(items, item)
				p.Advance()
				continue
			}
		} else {
			item = ast.EnumItem{Identifier: p.Expect(lexer.Identifier).Source}
		}
		if p.CurrentTokenKind() == lexer.Equal {
			p.Advance()
			item.Value = p.ParseExpression(PrimaryBindingPower)
		}
		items = append(items, item)
		if p.CurrentTokenKind() == lexer.EndOfStatement {
			p.Advance()
			continue
		}
		if p.IsNotCurrentlyEndOr(lexer.RightCurlyBrace) {
			p.Expect(lexer.Stroke)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.EnumDeclaration{Identifier: typeName, Values: items}
}

func (p *Parser) ParseStruct(typeName, firstField string, inherited []ast.Type) ast.StructDeclaration {
	var (
		isFirst = true
		fields  []ast.StructField
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		var field ast.StructField
		if isFirst {
			// First is currently at colon
			field = ast.StructField{Identifier: firstField}
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
	f.Identifier = p.Expect(lexer.Identifier).Source

	// Struct receiver
	// 	func Person.greet()
	if p.CurrentTokenKind() == lexer.Dot {
		p.Advance()
		f.Struct = ast.TypeAlias{Identifier: f.Identifier}
		f.Identifier = p.Expect(lexer.Identifier).Source
	}
	// Generic:
	//	func get<T, U>(a: T, b: [U]) -> T
	// Can't be assigned, only inferred
	if p.CurrentTokenKind() == lexer.LessThan {
		generics := []string{}
		p.Advance()
		for p.WhileNot(lexer.GreaterThan) {
			generics = append(generics, p.Expect(lexer.Identifier).Source)
			if p.CurrentTokenKind() != lexer.GreaterThan {
				p.Expect(lexer.Comma)
			}
		}
		p.Expect(lexer.GreaterThan)
		f.GenericParams = generics
	}
	// Params
	p.Expect(lexer.LeftParenthesis)
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		param := ast.FunctionParam{
			Identifier: p.Expect(lexer.Identifier).Source,
		}
		// Optional label:
		// 	func replace(src, with replacement: String)
		if p.CurrentTokenKind() == lexer.Identifier {
			param.Label = param.Identifier
			param.Identifier = p.Expect(lexer.Identifier).Source
		}
		// Parse type: still allow trailing type (example above)
		if p.CurrentTokenKind() == lexer.Colon {
			p.Advance()
			param.Type = p.ParseType(DefaultTypeBindingPower)
		}
		// Default value:
		// 	func List.join(by by: String = ", ")
		if p.CurrentTokenKind() == lexer.Equal {
			p.Advance()
			param.Default = p.ParseExpression(DefaultBindingPower)
		}

		f.Parameters = append(f.Parameters, param)
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)

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
		f.Expression = p.ParseExpression(DefaultBindingPower)
	}
	return f
}

func (p *Parser) ParseAttribute() (d ast.Attribute) {
	p.Expect(lexer.At)
	d.Decorator = p.Expect(lexer.Identifier).Source
	if p.CurrentTokenKind() == lexer.LeftParenthesis {
		call := p.ParseCallExpression(nil, CallBindingPower)
		d.Args = call.Args
	}
	return d
}

func (p *Parser) ParsePublicModifier() ast.Statement {
	p.Expect(lexer.Public)
	stmt := p.ParseStatement()
	if pub, ok := stmt.(ast.Publicizable); ok {
		pub.Publicize() // Set Public to true
	} else {
		p.Error(errors.Node(errors.ErrInvalidPublic, stmt))
	}
	return stmt
}
