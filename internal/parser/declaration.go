package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
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
		valType   ast.Type
		arrow     lexer.Token
		isEnum    bool
	)
	// Check if '->' was used on non-enum
	defer func() {
		if valType != nil && !isEnum {
			p.Error(errors.Token(errors.ErrInvalidArrow, arrow))
		}
	}()
	// Enum value type
	maybeParseArrow := func() {
		if p.CurrKind() == lexer.Arrow {
			arrow = p.Advance()
			valType = p.ParseType(DefaultTypeBindingPower)
		}
	}
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
	case lexer.Colon, lexer.Arrow:
		if p.CurrKind() == lexer.Colon {
			// Inherited struct, interface, or enum
			inherited = p.parseInheritedTypes()
		}
		maybeParseArrow()
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
		attrs := p.maybeParseAttributes()
		if p.CurrKind() == lexer.Dot || p.PeekKind() == lexer.LeftParenthesis {
			isEnum = true
			return p.ParseEnum(name, nil, inherited, valType, attrs)
		}
		return p.ParseStruct(name, inherited, attrs)
	case lexer.LessThan:
		var (
			lt       = p.Advance().Position
			generics []ast.Identifier
			res      ast.TypeDeclaration
		)
		parseSeries(p, &generics, p.ParseIdentifier, lexer.GreaterThan, lexer.Comma, false)
		gt := p.lastTokEnd()
		if p.CurrKind() == lexer.Colon {
			inherited = p.parseInheritedTypes()
		}
		maybeParseArrow()
		p.Expect(lexer.LeftCurlyBrace)
		if isIntf {
			res = p.ParseInterface(name, inherited)
		} else {
			attrs := p.maybeParseAttributes()
			if p.CurrKind() == lexer.Dot || p.PeekKind() == lexer.LeftParenthesis {
				isEnum = true
				return p.ParseEnum(name, generics, inherited, valType, attrs)
			}
			res = p.ParseStruct(name, inherited, attrs)
		}
		// Enum already returned
		p.Error(errors.Range(errors.ErrInvalidGenericType, ranges.Range{lt, gt}))
		return res
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

func (p *Parser) parseInheritedTypes() (inherited []ast.Type) {
	p.Advance() // :
	for p.HasTokens() {
		// Valid: Alias or generic (others still parsed)
		inherited = append(inherited, p.ParseType(DefaultTypeBindingPower))
		if p.CurrKind() != lexer.Comma {
			break
		}
		p.Expect(lexer.Comma)
	}
	return
}

func (p *Parser) ParseEnum(
	typeName ast.Identifier, generics []ast.Identifier,
	inherited []ast.Type, valType ast.Type, attrs []*ast.Attribute,
) *ast.EnumDeclaration {
	var (
		items   []*ast.EnumItem
		itemMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		if a := p.maybeParseAttributes(); len(a) > 0 {
			attrs = a
		}
		p.ExpectNoAdvance(lexer.Dot)
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
		if p.isEqual() {
			p.Advance()
			item.Value = p.ParseExpression(MemberBindingPower)
			switch c := p.Curr(); c.Kind {
			case lexer.Dot:
				if c.Line != item.Value.GetRange().End.Line {
					break
				}
				fallthrough
			default:
				item.Value, _ = p.handleLED(c.Kind, item.Value, ExpressionBindingPower)
			case lexer.RightCurlyBrace, lexer.EndOfStatement, lexer.At, lexer.Comma:
			}
		}
		items = append(items, markStartEndPos(p, item, id.Position))
		attrs = nil
		switch c := p.Curr(); c.Kind {
		case lexer.RightCurlyBrace:
		case lexer.EndOfStatement, lexer.Comma:
			p.Advance()
			continue
		case lexer.Dot:
			if c.Line > item.Range.End.Line {
				continue // No EOS before '.' in next item
			}
			fallthrough
		default:
			p.Expect(lexer.Comma, lexer.EndOfStatement)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return &ast.EnumDeclaration{
		Identifier: typeName,
		Generics:   generics,
		Inherited:  inherited,
		ValueType:  valType,
		Values:     items,
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

func (p *Parser) ParseStruct(
	typeName ast.Identifier, inherited []ast.Type, attrs []*ast.Attribute,
) *ast.StructDeclaration {
	fieldMap := make(map[string]struct{})
	str := &ast.StructDeclaration{Identifier: typeName, InheritedTypes: inherited}

	parseSeries(p, &str.Fields, func() *ast.StructField {
		if a := p.maybeParseAttributes(); a != nil {
			attrs = a
		}
		f := &ast.StructField{Attributes: attrs}
		// Keys
		parseSeries(p, &f.Names, func() ast.Identifier {
			name := p.ParseMapIdentifier(0)
			if _, ok := fieldMap[name.Name]; ok {
				err := errors.Node(errors.ErrRedeclaredField, name)
				err.SetParam("kind", "struct")
				// TODO: original position in error
				// maybe store index as fieldMap value
				p.Error(err)
				return name
			}
			fieldMap[name.Name] = struct{}{}
			return name
		}, 0, lexer.Comma, false)
		// Type
		if isAssignment(p.CurrKind()) {
			// Default value without explicit type
			p.Error(errors.Token(errors.ErrRequiredStructFieldType, p.Curr()))
		} else {
			p.ExpectErrorNoAdvance(
				errors.Slice(errors.ErrRequiredStructFieldType, f.Names),
				lexer.Colon,
			)
			f.Type = p.ParseType(DefaultTypeBindingPower)
		}
		// Default value
		if p.isEqual() {
			p.Advance()
			f.Value = p.ParseExpression(ExpressionBindingPower)
		}
		f.Range.Start = f.Names[0].Position
		attrs = nil
		return f
	}, lexer.RightCurlyBrace, lexer.EndOfStatement, true)
	return str
}

// Similar to [*Parser.ParseTupleType]
func (p *Parser) ParseInterfaceFuncParams() (params []*ast.MethodTypeParam) {
	var names [][2]ast.Identifier
	var isType, hasColon, hasLabel bool
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		if k := p.CurrKind(); !isType && (hasLabel || isValidIdentOrDiscard(k) ||
			isValidIdentOrDiscard(p.PeekKind())) && p.PeekKind() != lexer.Dot {
			var name [2]ast.Identifier
			if hasLabel || isValidIdentifier(p.PeekKind()) {
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

func (p *Parser) ParseInterface(
	typeName ast.Identifier, inherited []ast.Type,
) *ast.InterfaceDeclaration {
	fieldMap := make(map[string]struct{})
	intf := &ast.InterfaceDeclaration{Identifier: typeName, InheritedTypes: inherited}

	parseSeries(p, &intf.Items, func() *ast.InterfaceItem {
		f := &ast.InterfaceItem{Attributes: p.maybeParseAttributes()}
		// Names
		parseSeries(p, &f.Keys, func() ast.Identifier {
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
			if len(f.Keys) > 1 {
				// Invalid: x, y, z()
				p.Error(errors.Slice(errors.ErrIntfMultiKeyMethod, f.Keys))
			}
			fn := &ast.MethodType{
				BaseNode: ast.BaseNode{Range: ranges.Range{
					Start: p.Advance().Position, // (
				}},
				Parameters: p.ParseInterfaceFuncParams(),
			}
			if p.CurrKind() == lexer.Arrow {
				p.Advance()
				fn.ReturnType = p.ParseType(DefaultTypeBindingPower)
			}
			fn.Range.End = p.lastTokEnd()
			f.Value = fn
		} else {
			p.ExpectNoAdvance(lexer.Colon)
			f.Value = p.ParseType(DefaultTypeBindingPower)
		}
		if c := p.Curr(); c.Kind == lexer.Equal || c.Kind == lexer.ColonEqual {
			p.Error(errors.Token(errors.ErrIntfDefaultValue, c))
			p.ParseExpression(DefaultBindingPower) // Just to skip the expression
		}
		f.Range.Start = f.Keys[0].Position
		return f
	}, lexer.RightCurlyBrace, lexer.EndOfStatement, true)
	return intf
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
		if p.CurrKind() != lexer.Colon { // :
			p.Expect(lexer.Colon) // Report error
		} else {
			p.Advance()
			rec = p.ParseIdentifier()        // Struct name
			p.Expect(lexer.RightParenthesis) // )
		}
		if p.CurrKind() != lexer.Dot {
			p.Error(errors.Token(errors.ErrFuncDotAfterSelf, p.Curr()))
			// Just set it for error tolerance
			str := rec
			f.Struct = &str
			rec = p.ParseMapIdentifier(0) // Goes to f.Identifier = rec
		}
		// Otherwise, p.CurrKind() == lexer.Dot
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
	if p.isEqual() {
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
			switch peek := p.PeekKind(); peek {
			case lexer.Colon, lexer.Equal, lexer.ColonEqual,
				lexer.Comma, lexer.RightParenthesis:
			default:
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
		if p.isEqual() {
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
	} else if p.isEqual() {
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
