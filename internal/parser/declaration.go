package parser

import (
	"cmp"

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
	tryParseArrow := func() {
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
		tryParseArrow()
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
		attrs := p.tryParseAttributes()
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
		tryParseArrow()
		p.Expect(lexer.LeftCurlyBrace)
		if isIntf {
			res = p.ParseInterface(name, inherited)
		} else {
			attrs := p.tryParseAttributes()
			if p.CurrKind() == lexer.Dot || p.PeekKind() == lexer.LeftParenthesis {
				isEnum = true
				return p.ParseEnum(name, generics, inherited, valType, attrs)
			}
			res = p.ParseStruct(name, inherited, attrs)
		}
		// Enum already returned
		p.Error(errors.Range(errors.ErrInvalidGenericType, ranges.Range{lt, gt}))
		return res
	case lexer.Newline:
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
		if p.CurrKind() != lexer.Newline {
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
		if a := p.tryParseAttributes(); len(a) > 0 {
			attrs = a
		}
		p.ExpectNoAdvance(lexer.Dot)
		id := p.ParseMapIdentOrDiscard(0)

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
		if p.isEqual(p.Curr()) {
			p.Advance()
			item.Value = p.ParseExpressionFilter(excludeIf(lexer.Dot),
				MemberBindingPower, allowIfSameLine,
			)
		}
		items = append(items, markStartEndPos(p, item, id.Position))
		attrs = nil
		switch c := p.Curr(); c.Kind {
		case lexer.RightCurlyBrace:
		case lexer.Newline, lexer.Comma:
			p.Advance()
			continue
		case lexer.Dot:
			if c.Line > item.Range.End.Line {
				continue // No EOS before '.' in next item
			}
			fallthrough
		default:
			p.Expect(lexer.Comma, lexer.Newline)
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

func (p *Parser) tryParseAttributes() (attrs []*ast.Attribute) {
	for p.HasTokens() && p.CurrKind() == lexer.At {
		attrs = append(attrs, p.ParseAttribute())
		if curr := p.CurrKind(); curr == lexer.Newline {
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
		if a := p.tryParseAttributes(); a != nil {
			attrs = a
		}
		f := &ast.StructField{Attributes: attrs}
		// Keys
		parseSeries(p, &f.Names, func() ast.Identifier {
			name := p.ParseMapIdentOrDiscard(0)
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
		if p.isEqual(p.Curr()) {
			p.Advance()
			f.Value = p.ParseExpression(ExpressionBindingPower)
		}
		f.Range.Start = f.Names[0].Position
		attrs = nil
		return f
	}, lexer.RightCurlyBrace, lexer.Newline, true)
	return str
}

func (p *Parser) ParseInterface(
	typeName ast.Identifier, inherited []ast.Type,
) *ast.InterfaceDeclaration {
	fieldMap := make(map[string]struct{})
	intf := &ast.InterfaceDeclaration{Identifier: typeName, InheritedTypes: inherited}

	parseSeries(p, &intf.Items, func() *ast.InterfaceItem {
		f := &ast.InterfaceItem{
			Attributes: p.tryParseAttributes(),
			TypePair:   &ast.TypePair{},
		}
		// Names
		parseSeries(p, &f.Keys, func() ast.Identifier {
			if p.CurrKind() == lexer.Underscore {
				p.ErrorLabelled(errors.Token(errors.ErrDiscardIntfField, p.Curr()), "Remove the field")
			}
			name := p.ParseMapIdentOrDiscard(0)
			if _, ok := fieldMap[name.Name]; ok {
				err := errors.Node(errors.ErrRedeclaredField, name)
				err.SetParam("kind", "interface")
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
				Parameters: p.parseMethodParams(),
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
	}, lexer.RightCurlyBrace, lexer.Newline, true)
	return intf
}

func (p *Parser) ParseFuncDeclaration() ast.Statement {
	p.Expect(lexer.Func)
	f := &ast.FunctionDeclaration{}

	// func (p: Parser).
	if p.CurrKind() == lexer.LeftParenthesis {
		// Method declaration with receiver alias
		// 	func (t: Type).method()
		p.Advance() // (
		if p.PeekKind() == lexer.Colon {
			if p.CurrKind() == lexer.Underscore {
				p.Error(errors.Token(errors.ErrSelfNameDiscard, p.Curr()))
			}
			f.SelfName = new(p.ParseIdentOrDiscard())
			p.Expect(lexer.Colon)
		}
		f.Struct = new(p.ParseIdentifier()) // TODO: change type of f.Struct to allow types
		p.Expect(lexer.RightParenthesis)    // )
		p.ExpectErrorCodeNoAdvance(errors.ErrFuncDotAfterSelf, lexer.Dot)
		f.Identifier = p.ParseMapIdentifier(0)
	} else if p.PeekKind() == lexer.Dot {
		// Method declaration
		// 	func Type.method()
		f.Struct = new(p.ParseIdentifier())
		p.Expect(lexer.Dot)
		f.Identifier = p.ParseMapIdentifier(0)
	} else {
		// Normal function declaration
		// 	func fn()
		f.Identifier = p.ParseIdentOrDiscard()
	}

	// Generic:
	//	func get<T, U>(a: T, b: [U]) -> T
	// Can't be assigned, only inferred
	if p.CurrKind() == lexer.LessThan {
		p.Advance()
		parseSeries(p, &f.GenericParams, p.ParseIdentifier,
			lexer.GreaterThan, lexer.Comma, false,
		)
		if len(f.GenericParams) == 0 {
			p.Error(errors.Token(errors.ErrEmptyGeneric, p.PeekBehind()))
		}
	}

	// Function alias
	// 	func fn = otherFn
	if p.isEqual(p.Curr()) {
		return p.ParseFuncAlias(f)
	}

	// Params
	p.Expect(lexer.LeftParenthesis)
	parseSeries(p, &f.Parameters, p.parseFuncParam, lexer.RightParenthesis, lexer.Comma, false)

	// Return type: after the arrow. Not required if returns Nothing
	if p.CurrKind() == lexer.Arrow {
		p.Advance()
		f.ReturnType = p.ParseType(DefaultTypeBindingPower)
	}

	// Body: Externally implemented functions may not have a body
	//	@external(js: "./date.js", name: "now")
	// 	func Date.now() -> Date
	if p.CurrKind() == lexer.LeftCurlyBrace {
		f.Body = p.ParseBlock()
	} else if p.isEqual(p.Curr()) {
		// Expression body
		// 	func fn() = 2
		p.Advance()
		f.Expression = p.ParseExpression(ExpressionBindingPower)
	}
	return f
}

func (p *Parser) parseFuncParam() *ast.FunctionParam {
	param := &ast.FunctionParam{}
	param.Range.Start = p.Curr().Position

	// Trailing params
	parseSeries(p, &param.Names, func() *ast.IdentifierPair {
		key := &ast.IdentifierPair{}
		// Optional label:
		// 	func replace(src, with replacement: String)
		if k := p.PeekKind(); isValidIdentOrDiscard(k) || k == lexer.Newline {
			key.Label = p.ParseMapIdentifier(isLabel)
			if k == lexer.Newline {
				p.Advance()
			}
		}
		// Normal identifier
		key.Name = p.ParseIdentOrDiscard()
		markStartEndPos(p, key, cmp.Or(key.Label.Position, key.Name.Position))
		return key
	}, 0, lexer.Comma, true)

	// Type
	p.ExpectErrorCodeNoAdvance(errors.ErrMissingFuncParamType, lexer.Colon)
	if !isAssignment(p.PeekBehind().Kind) {
		param.Type = p.ParseType(DefaultTypeBindingPower)
	}

	// Default value:
	// 	func List.join(by by: String = ", ")
	if p.isEqual(p.Curr()) {
		if len(param.Names) > 1 {
			p.ErrorLabelled(
				errors.Range(errors.ErrChainedDefault, ranges.Range{
					Start: param.Names[len(param.Names)-1].Range.Start,
					End:   param.Type.GetRange().End,
				}), "Unchain this parameter",
			)
		}
		p.Advance()
		param.Default = p.ParseExpression(ExpressionBindingPower)
	}
	return markEndPos(p, param)
}

func (p *Parser) ParseFuncAlias(f *ast.FunctionDeclaration) *ast.FuncAliasDeclaration {
	beforeEqual := p.Index - 1
	p.Advance() // =
	if f.GenericParams != nil {
		p.Error(errors.Node(errors.ErrGenericInFuncAlias, f.Identifier))
	}
	if f.SelfName != nil {
		p.Error(errors.Node(errors.ErrSelfLabelInFuncAlias, f.Identifier))
	}
	target := p.ParseExpression(ExpressionBindingPower)
	switch target := target.(type) {
	case *ast.Symbol:
	case *ast.IndexExpression:
		if target.Computed {
			p.Error(errors.Node(errors.ErrComputedFuncAlias, target))
		}
		// LHS checked at analysis-time
	default:
		err := errors.Node(errors.ErrNonNameFuncAlias, target)
		err.HintWithDiff(
			"Or, did you mean to define a new function? Add parentheses after the function name.",
			&errors.Diff{Edits: []errors.DiffEdit{errors.AddedString{
				Position: p.Tokens[beforeEqual].End(),
				String:   "()",
			}}},
		)
		p.Error(err)
	}
	return &ast.FuncAliasDeclaration{
		Identifier: f.Identifier,
		Struct:     f.Struct,
		Target:     target,
	}
}

func (p *Parser) ParseAttribute() *ast.Attribute {
	p.Expect(lexer.At)
	d := &ast.Attribute{}
	p.flags |= isAttribute
	d.Decorator = p.ParseIdentifier()
	if p.CurrKind() == lexer.LeftParenthesis {
		call := p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis))
		d.Args = call.Args
	}
	p.flags &^= isAttribute
	return d
}
