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
		name      = p.ParseIdentifier()
		inherited []ast.Type
	)

	switch p.CurrentTokenKind() {
	case lexer.Equal:
		if isIntf {
			p.Error(errors.ExpectedToken(lexer.LeftCurlyBrace, p.CurrentToken()))
		}
		// Type alias
		p.Advance()
		return &ast.TypeAliasDeclaration{
			Identifier: name,
			Type:       p.ParseType(DefaultTypeBindingPower),
		}
	case lexer.Colon:
		// Inherited struct, interface, or enum
		p.Advance()
		for p.WhileNotEndOr(lexer.LeftCurlyBrace) {
			// Alias or generic
			inherited = append(inherited, p.ParseType(PrimaryTypeBindingPower))
			if p.CurrentTokenKind() != lexer.LeftCurlyBrace {
				p.Expect(lexer.Comma)
			}
		}
		fallthrough
	case lexer.LeftCurlyBrace:
		// Struct, enum, or interface
		p.Expect(lexer.LeftCurlyBrace)
		if isIntf {
			return p.ParseInterface(name, inherited)
		}
		if p.CurrentTokenKind() == lexer.RightCurlyBrace {
			// Empty struct
			p.Advance()
			return &ast.StructDeclaration{
				Identifier:     name,
				InheritedTypes: inherited,
			}
		}
		if p.CurrentTokenKind() == lexer.Dot {
			return p.ParseEnum(name, inherited)
		}
		return p.ParseStruct(name, inherited)
	case lexer.EndOfStatement:
		// Type tag if interface
		if isIntf {
			return &ast.InterfaceDeclaration{
				Tag:            true,
				Identifier:     name,
				InheritedTypes: inherited,
			}
		}
		fallthrough
	default:
		// Some other token or unassigned type (if EOS)
		p.Error(errors.Token(errors.ErrExpectedTypeAssignment, p.CurrentToken()))
		p.Advance()
		return &ast.TypeAliasDeclaration{
			Identifier: name,
			Type:       &ast.BadExpression{Token: p.CurrentToken().Kind},
		}
	}
}

func (p *Parser) ParseEnum(typeName ast.Identifier, inherited []ast.Type) *ast.EnumDeclaration {
	var (
		items   []*ast.EnumItem
		itemMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		p.Expect(lexer.Dot)
		id := p.ParseIdentifier()

		// Check if exists
		if _, ok := itemMap[id.Name]; ok {
			err := errors.Node(errors.ErrRedeclaredField, id)
			err.SetParam("kind", "enum")
			p.Error(err)
		}
		itemMap[id.Name] = struct{}{}

		item := &ast.EnumItem{Identifier: id}
		if p.CurrentTokenKind() == lexer.LeftParenthesis {
			p.Advance()
			parseSeries(p, &item.Parameters,
				func() ast.Type { return p.ParseType(DefaultTypeBindingPower) },
				lexer.RightParenthesis, lexer.Comma, false,
			)
		}
		if p.isEqualOrColonEqualAndError() {
			p.Advance()
			item.Value = p.ParseExpression(PrimaryBindingPower)
		}
		markStartEndPos(p, item, id.Position)
		items = append(items, item)
		if c := p.CurrentToken(); c.Kind == lexer.EndOfStatement {
			p.Advance()
			continue 
		} else if c.Kind == lexer.Dot && c.Line > item.Range.End.Line {
			continue // No EOS before '.' in next item
		}
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return &ast.EnumDeclaration{
		Identifier: typeName,
		Values:     items,
		Inherited:  inherited,
	}
}

func (p *Parser) ParseStruct(typeName ast.Identifier, inherited []ast.Type) *ast.StructDeclaration {
	var (
		fields   []*ast.StructField
		fieldMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		field := &ast.StructField{}
		parseSeries(p, &field.Names, func() ast.Identifier {
			name := p.ParseMapIdentifier(false)
			if _, ok := fieldMap[name.Name]; ok {
				err := errors.Node(errors.ErrRedeclaredField, name)
				err.SetParam("kind", "struct")
				p.Error(err)
			}
			fieldMap[name.Name] = struct{}{}
			return name
		}, 0, lexer.Comma, false)
		// Type
		p.ExpectError(errors.Token(
			errors.ErrRequiredStructFieldType, p.CurrentToken(),
		), lexer.Colon)
		field.Type = p.ParseType(DefaultTypeBindingPower)
		// Default value
		if p.isEqualOrColonEqualAndError() {
			p.Advance()
			field.Value = p.ParseExpression(ExpressionBindingPower)
		}
		markStartEndPos(p, field, field.Names[0].Position)
		fields = append(fields, field)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return &ast.StructDeclaration{
		Identifier:     typeName,
		InheritedTypes: inherited,
		Fields:         fields,
	}
}

func (p *Parser) ParseInterface(typeName ast.Identifier, inherited []ast.Type) *ast.InterfaceDeclaration {
	var (
		fields   []*ast.TypePair
		fieldMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		field := &ast.TypePair{}
		parseSeries(p, &field.Keys, func() ast.Identifier {
			name := p.ParseMapIdentifier(false)
			if _, ok := fieldMap[name.Name]; ok {
				err := errors.Node(errors.ErrRedeclaredField, name)
				err.SetParam("kind", "struct")
				p.Error(err)
			}
			fieldMap[name.Name] = struct{}{}
			return name
		}, 0, lexer.Comma, false)
		// Type
		if p.CurrentTokenKind() == lexer.LeftParenthesis {
			// Parse function: #{ Kind() -> string }
			fn := &ast.MethodType{}
			p.Advance() // (
			parseSeries(p, &fn.Parameters, func() *ast.MethodTypeParam {
				var label, name ast.Identifier
				if p.Peek().Kind == lexer.Identifier {
					// Label: fib(length length: Int)
					label = p.ParseMapIdentifier(false)
					name = p.ParseIdentifier()
					p.Expect(lexer.Colon)
				} else if p.Peek().Kind == lexer.Colon {
					// Declared wih a name
					name = p.ParseIdentifier()
					p.Expect(lexer.Colon)
				}
				// Type
				return &ast.MethodTypeParam{
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
		if c := p.CurrentToken(); c.Kind == lexer.Equal || c.Kind == lexer.ColonEqual {
			p.Error(errors.Token(errors.ErrInterfaceDefaultValue, c))
			p.ParseExpression(DefaultBindingPower) // Just to skip the expression
		}
		markStartEndPos(p, field, field.Keys[0].Position)
		fields = append(fields, field)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return &ast.InterfaceDeclaration{
		Identifier:     typeName,
		InheritedTypes: inherited,
		Fields:         fields,
	}
}

func (p *Parser) ParseFuncDeclaration() ast.Statement {
	p.Expect(lexer.Func)
	f := &ast.FunctionDeclaration{}
	rec := p.ParseIdentifier()

	// Struct receiver
	// 	func Person.greet()
	if p.CurrentTokenKind() == lexer.Dot {
		f.Struct = rec
		p.Advance() // .
		f.Identifier = p.ParseMapIdentifier(false)
	} else {
		f.Identifier = rec
	}
	// Generic:
	//	func get<T, U>(a: T, b: [U]) -> T
	// Can't be assigned, only inferred
	if p.CurrentTokenKind() == lexer.LessThan {
		p.Advance()
		parseSeries(p, &f.GenericParams, p.ParseIdentifier, lexer.GreaterThan, lexer.Comma, false)
		if len(f.GenericParams) == 0 {
			p.Error(errors.Token(errors.ErrEmptyGeneric, p.PeekBehind()))
		}
	}
	// Function alias
	if p.isEqualOrColonEqualAndError() {
		p.Advance()
		if f.GenericParams != nil {
			p.Error(errors.Node(errors.ErrGenericInFuncAlias, f.Identifier))
		}
		alias := p.ParseExpression(ExpressionBindingPower)
		switch alias := alias.(type) {
		case *ast.Symbol, *ast.IndexExpression:
		default:
			// Note: computed member expressions not validated
			p.Error(errors.Node(errors.ErrNonNameFuncAlias, alias))
		}
		return &ast.FuncAliasDeclaration{
			Identifier: f.Identifier,
			Struct:     f.Struct,
			Alias:      alias,
		}
	}
	// Params
	p.Expect(lexer.LeftParenthesis)
	parseSeries(p, &f.Parameters, func() *ast.FunctionParam {
		param := &ast.FunctionParam{}
		param.Range.Start = p.CurrentToken().Position

		// Trailing type params
		parseSeries(p, &param.Names, func() *ast.FunctionParamName {
			key := &ast.FunctionParamName{}
			if peek := p.Peek().Kind; peek == lexer.Identifier ||
				(peek != lexer.Colon && peek != lexer.Equal && peek != lexer.ColonEqual) {
				// Optional label:
				// 	func replace(src, with replacement: String)
				key.Label = p.ParseMapIdentifier(false)
				if peek == lexer.EndOfStatement {
					p.Advance()
				}
			}
			// Normal identifier
			key.Identifier = p.ParseIdentifier()
			return key
		}, 0, lexer.Comma, true)

		// Type
		p.ExpectErrorCode(errors.ErrMissingFuncParamType, lexer.Colon)
		param.Type = p.ParseType(DefaultTypeBindingPower)
		// Default value:
		// 	func List.join(by by: String = ", ")
		if p.isEqualOrColonEqualAndError() {
			p.Advance()
			param.Default = p.ParseExpression(ExpressionBindingPower)
		}
		markEndPos(p, param)
		f.Parameters = append(f.Parameters, param)
		return param
	}, lexer.RightParenthesis, lexer.Comma, false)

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
	} else if p.isEqualOrColonEqualAndError() {
		p.Advance()
		f.Expression = p.ParseExpression(ExpressionBindingPower)
	}
	return f
}

func (p *Parser) ParseAttribute() *ast.Attribute {
	p.Expect(lexer.At)
	d := &ast.Attribute{}
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
	switch stmt.(type) {
	case *ast.EnumDeclaration, *ast.FunctionDeclaration,
		*ast.InterfaceDeclaration, ast.TypeDeclaration,
		*ast.StructDeclaration, *ast.TypeAliasDeclaration,
		*ast.VariableDeclaration, *ast.FuncAliasDeclaration:
		return &ast.PublicDeclaration{Declaration: stmt}
	default:
		p.Error(errors.Node(errors.ErrInvalidPublic, stmt))
		return &ast.BadExpression{Value: stmt}
	}
}
