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
	isIntf := p.CurrKind() == lexer.Hash
	if isIntf {
		p.Advance()
	}
	var (
		name      = p.ParseIdentOrDiscard()
		inherited []ast.Type
	)

	switch p.CurrKind() {
	case lexer.Equal:
		if isIntf {
			p.Error(errors.ExpectedToken(lexer.LeftCurlyBrace, p.Curr()))
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
			if p.CurrKind() != lexer.LeftCurlyBrace {
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
		if p.CurrKind() == lexer.RightCurlyBrace {
			// Empty struct
			p.Advance()
			return &ast.StructDeclaration{
				Identifier:     name,
				InheritedTypes: inherited,
			}
		}
		if p.CurrKind() == lexer.Dot {
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
		p.Error(errors.Token(errors.ErrExpectedTypeAssignment, p.Curr()))
		if p.CurrKind() != lexer.EndOfStatement {
			p.Advance()
		}
		return &ast.TypeAliasDeclaration{
			Identifier: name,
			Type:       &ast.BadExpression{Token: p.Curr().Kind},
		}
	}
}

func (p *Parser) ParseEnum(typeName ast.Identifier, inherited []ast.Type) *ast.EnumDeclaration {
	var (
		items   []*ast.EnumItem
		itemMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		attrs := p.maybeParseAttributes()
		p.Expect(lexer.Dot)
		id := p.ParseIdentifier()

		// Check if exists
		if _, ok := itemMap[id.Name]; ok {
			err := errors.Node(errors.ErrRedeclaredField, id)
			err.SetParam("kind", "enum")
			p.Error(err)
		}
		itemMap[id.Name] = struct{}{}

		item := &ast.EnumItem{Identifier: id, Attributes: attrs}
		if p.CurrKind() == lexer.LeftParenthesis {
			item.Parameters = p.ParseTupleType().Values
		}
		if p.isEqualOrColonEqualAndError() {
			p.Advance()
			item.Value = p.ParseExpression(ExpressionBindingPower)
		}
		markStartEndPos(p, item, id.Position)
		items = append(items, item)
		if c := p.Curr(); c.Kind == lexer.EndOfStatement {
			p.Advance()
			continue
		} else if c.Kind == lexer.Dot && c.Line > item.Range.End.Line {
			continue // No EOS before '.' in next item
		} else if c.Kind != lexer.RightCurlyBrace {
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

func (p *Parser) maybeParseAttributes() (attrs []*ast.Attribute) {
	for p.HasTokens() && p.CurrKind() == lexer.At {
		attrs = append(attrs, p.ParseAttribute())
		if curr := p.CurrKind(); curr == lexer.EndOfStatement {
			p.Advance()
		} else if curr != lexer.At {
			break
		}
	}
	return attrs
}

func (p *Parser) ParseStruct(typeName ast.Identifier, inherited []ast.Type) *ast.StructDeclaration {
	var (
		fields   []*ast.StructField
		fieldMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		field := &ast.StructField{}
		field.Attributes = p.maybeParseAttributes()
		parseSeries(p, &field.Names, func() ast.Identifier {
			name := p.ParseMapIdentifier(0)
			if _, ok := fieldMap[name.Name]; ok {
				err := errors.Node(errors.ErrRedeclaredField, name)
				err.SetParam("kind", "struct")
				p.Error(err)
			}
			fieldMap[name.Name] = struct{}{}
			return name
		}, 0, lexer.Comma, false)
		// Type
		if isAssignment(p.CurrKind()) {
			p.Error(errors.Token(errors.ErrRequiredStructFieldType, p.Curr()))
		} else {
			p.ExpectErrorNoAdvance(
				errors.Slice(errors.ErrRequiredStructFieldType, field.Names),
				lexer.Colon,
			)
			field.Type = p.ParseType(DefaultTypeBindingPower)
		}
		// Default value
		if p.isEqualOrColonEqualAndError() {
			p.Advance()
			field.Value = p.ParseExpression(ExpressionBindingPower)
		}
		markStartEndPos(p, field, field.Names[0].Position)
		fields = append(fields, field)
		if p.CurrKind() != lexer.RightCurlyBrace {
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

// Similar to [*Parser.ParseTupleType]
func (p *Parser) ParseInterfaceFuncParams() (params []*ast.MethodTypeParam) {
	var names [][2]ast.Identifier
	var isType, hasColon, hasLabel bool
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		if k := p.CurrKind(); !isType && (hasLabel || isValidIdentOrDiscard(k) ||
			isValidIdentOrDiscard(p.Peek().Kind)) && p.Peek().Kind != lexer.Dot {
			var name [2]ast.Identifier
			if hasLabel || isValidIdentifier(p.Peek().Kind) {
				name[0] = p.ParseMapIdentifier(isLabel)
			}
			name[1] = p.ParseValidIdent()
			names = append(names, name)
			if p.CurrKind() == lexer.Colon {
				if isType {
					p.Error(errors.Token(errors.ErrMixTypeTupleLabels, p.Curr()))
				}
				p.Advance() // :
				params = append(params, &ast.MethodTypeParam{
					Names: names, Type: p.ParseType(DefaultTypeBindingPower),
				})
				names = names[:0]
				hasColon = true
			}
		} else {
			isType = true
			t := p.ParseType(DefaultTypeBindingPower)
			if hasColon {
				p.Error(errors.Node(errors.ErrMixTypeTupleLabels, t))
			}
			params = append(params, &ast.MethodTypeParam{Type: t})
		}
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	if len(names) > 0 {
		// TODO: can improve error
		if peek := p.PeekBehind(); hasColon {
			p.Error(errors.Token(errors.ErrMixTypeTupleLabels, peek))
		} else if hasLabel {
			p.Error(errors.Token(errors.ErrMixTypeTupleLabels, peek))
		}
		for _, name := range names {
			params = append(params, &ast.MethodTypeParam{
				Type: &ast.TypeAlias{
					BaseNode:   name[1].BaseNode(),
					Identifier: name[1].Name,
				},
			})
		}
	}
	return params
}

func (p *Parser) ParseInterface(typeName ast.Identifier, inherited []ast.Type) *ast.InterfaceDeclaration {
	var (
		fields   []*ast.InterfaceItem
		fieldMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		field := &ast.InterfaceItem{Attributes: p.maybeParseAttributes()}
		parseSeries(p, &field.Keys, func() ast.Identifier {
			name := p.ParseMapIdentifier(0)
			if _, ok := fieldMap[name.Name]; ok {
				err := errors.Node(errors.ErrRedeclaredField, name)
				err.SetParam("kind", "struct")
				p.Error(err)
			}
			fieldMap[name.Name] = struct{}{}
			return name
		}, 0, lexer.Comma, false)
		// Type
		if p.CurrKind() == lexer.LeftParenthesis {
			// Parse function: #{ kind() -> String }
			if len(field.Keys) > 1 {
				// Invalid: x, y, z()
				p.Error(errors.Token(errors.ErrIntfMultiKeyMethod, p.Curr()))
			}
			fn := &ast.MethodType{} // TODO: position data?
			p.Advance()             // (
			fn.Parameters = p.ParseInterfaceFuncParams()
			if p.CurrKind() == lexer.Arrow {
				p.Advance()
				fn.ReturnType = p.ParseType(DefaultTypeBindingPower)
			}
			field.Value = fn
		} else {
			p.ExpectNoAdvance(lexer.Colon)
			field.Value = p.ParseType(DefaultTypeBindingPower)
		}
		if c := p.Curr(); c.Kind == lexer.Equal || c.Kind == lexer.ColonEqual {
			p.Error(errors.Token(errors.ErrInterfaceDefaultValue, c))
			p.ParseExpression(DefaultBindingPower) // Just to skip the expression
		}
		markStartEndPos(p, field, field.Keys[0].Position)
		fields = append(fields, field)
		if p.CurrKind() != lexer.RightCurlyBrace {
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
	var rec ast.Identifier
	// func (p: Parser)
	if p.CurrKind() == lexer.LeftParenthesis {
		p.Advance()                 // (
		self := p.ParseIdentifier() // self name
		f.SelfName = &self
		p.Expect(lexer.Colon)            // :
		rec = p.ParseIdentifier()        // Struct name
		p.Expect(lexer.RightParenthesis) // )
		if p.CurrKind() != lexer.Dot {
			p.Error(errors.Token(errors.ErrFuncDotAfterSelf, p.Curr()))
			// Just set it for error tolerance
			f.Struct = &rec
			f.Identifier = p.ParseMapIdentifier(0)
		}
	} else {
		rec = p.ParseIdentifier()
	}

	// Struct receiver
	// 	func Person.greet()
	if p.CurrKind() == lexer.Dot {
		f.Struct = &rec
		p.Advance() // .
		f.Identifier = p.ParseMapIdentifier(0)
	} else {
		f.Identifier = rec
	}
	// Generic:
	//	func get<T, U>(a: T, b: [U]) -> T
	// Can't be assigned, only inferred
	if p.CurrKind() == lexer.LessThan {
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
		param.Range.Start = p.Curr().Position

		// Trailing type params
		parseSeries(p, &param.Names, func() *ast.FunctionParamName {
			key := &ast.FunctionParamName{}
			if peek := p.Peek().Kind; isValidIdentOrDiscard(peek) ||
				(peek != lexer.Colon && peek != lexer.Equal && peek != lexer.ColonEqual) {
				// Optional label:
				// 	func replace(src, with replacement: String)
				key.Label = p.ParseMapIdentifier(isLabel)
				if peek == lexer.EndOfStatement {
					p.Advance()
				}
			}
			// Normal identifier
			key.Identifier = p.ParseIdentOrDiscard()
			return key
		}, 0, lexer.Comma, true)

		// Type
		p.ExpectErrorCode(errors.ErrMissingFuncParamType, lexer.Colon)
		if isAssignment(p.PeekBehind().Kind) {
			p.Backup()
		} else {
			param.Type = p.ParseType(DefaultTypeBindingPower)
		}
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
	if p.CurrKind() == lexer.Arrow {
		p.Advance()
		f.ReturnType = p.ParseType(DefaultTypeBindingPower)
	}

	// Body: Externally implemented functions may not have a body
	//	@external(js: "./date.js", name: "now")
	// 	func Date.now() -> Date
	if p.CurrKind() == lexer.LeftCurlyBrace {
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
	d.Decorator = p.ParseIdentifier()
	if p.CurrKind() == lexer.LeftParenthesis {
		call := p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis))
		d.Args = call.Args
	}
	p.isAttribute = false
	return d
}
