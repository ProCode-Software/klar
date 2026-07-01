package parser

import (
	"cmp"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
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
	name := p.ParseIdentOrDiscard()
	var inherited []ast.Type
	switch p.CurrKind() {
	case lexer.Equal, lexer.ColonEqual:
		_ = p.isEqual(p.Curr()) // Report an error on ':='
		if isIntf {
			p.Error(klarerrs.ExpectedToken(lexer.LeftCurlyBrace, p.Curr()))
		}
		// Type alias
		p.Advance()
		return &ast.TypeAliasDeclaration{
			Identifier: name,
			Type:       p.ParseType(DefaultTypeBindingPower),
		}
	case lexer.Colon:
		// Inherited struct, interface, tag, or enum
		inherited = p.parseInheritedTypes()
		if isIntf && p.CurrKind() == lexer.Newline {
			return &ast.TagDeclaration{
				Identifier:     name,
				InheritedTypes: inherited,
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
		attrs := p.tryParseAttributes()
		if p.CurrKind() == lexer.Dot || p.PeekKind() == lexer.LeftParenthesis {
			return p.ParseEnum(name, nil, inherited, attrs)
		}
		return p.ParseStruct(name, inherited, attrs)
	case lexer.LessThan:
		var (
			lt       = p.Curr().Position
			generics = p.tryParseGenericDecl()
			gt       = p.lastTokEnd()
			res      ast.TypeDeclaration
		)
		if p.CurrKind() == lexer.Colon {
			inherited = p.parseInheritedTypes()
		}
		p.Expect(lexer.LeftCurlyBrace)
		if isIntf {
			res = p.ParseInterface(name, inherited)
		} else {
			attrs := p.tryParseAttributes()
			if p.CurrKind() == lexer.Dot || p.PeekKind() == lexer.LeftParenthesis {
				return p.ParseEnum(name, generics, inherited, attrs)
			}
			res = p.ParseStruct(name, inherited, attrs)
		}
		// Enum already returned
		p.Error(klarerrs.Range(klarerrs.ErrInvalidGenericType, ranges.Range{lt, gt}))
		return res
	case lexer.Newline:
		// Type tag if interface
		if isIntf {
			return &ast.TagDeclaration{
				Identifier:     name,
				InheritedTypes: inherited,
			}
		}
		fallthrough
	default:
		// Some other token or unassigned type (if EOS)
		p.Error(klarerrs.Token(klarerrs.ErrExpectedTypeAssignment, p.Curr()))
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
	inherited []ast.Type, attrs []*ast.Attribute,
) *ast.EnumDeclaration {
	var (
		items   []*ast.EnumItem
		itemMap = make(map[string]struct{})
	)
	for p.WhileNot(lexer.RightCurlyBrace) {
		if a := p.tryParseAttributes(); len(a) > 0 {
			attrs = a
		}
		p.Expect(lexer.Dot, noAdvance)
		id := p.ParseMapIdentOrDiscard(0)

		// Check if exists
		if _, ok := itemMap[id.Name]; ok {
			err := klarerrs.Node(klarerrs.ErrRedeclaredField, id)
			err.SetParam("kind", "enum")
			p.Error(err)
		}
		itemMap[id.Name] = struct{}{}

		item := &ast.EnumItem{Identifier: id, Attributes: attrs}
		if p.CurrKind() == lexer.LeftParenthesis {
			item.Parameters = toTupleType(p.ParseTupleType())
		}
		if p.isEqual(p.Curr()) {
			p.Advance()
			item.Value = p.ParseExpressionFilter(
				excludeIf(lexer.Dot),
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
			p.ExpectOneOf(lexer.Comma, lexer.Newline)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return &ast.EnumDeclaration{
		Identifier: typeName,
		Generics:   generics,
		Inherited:  inherited,
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
	fieldMap := make(map[string]ranges.Range)
	str := &ast.StructDeclaration{Identifier: typeName, InheritedTypes: inherited}

	parseSeries(p, &str.Fields, func() *ast.StructField {
		if a := p.tryParseAttributes(); a != nil {
			attrs = a
		}
		f := &ast.StructField{Attributes: attrs}
		// Keys
		parseSeries(p, &f.Names, func() ast.Identifier {
			name := p.ParseMapIdentOrDiscard(0)
			if r, ok := fieldMap[name.Name]; ok {
				err := klarerrs.Node(klarerrs.ErrRedeclaredField, name)
				err.SetParam("kind", "struct")
				err.AddDetail("It was originally declared here", p.Options.File, r)
				p.Error(err)
			} else {
				fieldMap[name.Name] = name.Range()
			}
			return name
		}, 0, lexer.Comma, true)
		// Type
		if isAssignment(p.CurrKind()) {
			// Default value without explicit type
			p.Error(klarerrs.Token(klarerrs.ErrRequiredStructFieldType, p.Curr()))
		} else {
			p.Expect(
				lexer.Colon,
				expectError{klarerrs.Slice(klarerrs.ErrRequiredStructFieldType, f.Names)},
				noAdvance,
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
	type declared struct {
		rang   ranges.Range
		method bool
	}
	fieldMap := make(map[string]declared)
	intf := &ast.InterfaceDeclaration{Identifier: typeName, InheritedTypes: inherited}

	parseSeries(p, &intf.Items, func() *ast.InterfaceItem {
		f := &ast.InterfaceItem{
			Attributes: p.tryParseAttributes(),
			TypePair:   &ast.TypePair{},
		}
		// Names
		parseSeries(p, &f.Keys, func() ast.Identifier {
			if p.CurrKind() == lexer.Underscore {
				p.ErrorLabelled(klarerrs.Token(klarerrs.ErrDiscardIntfField, p.Curr()), "Remove the field")
			}
			name := p.ParseMapIdentOrDiscard(0)
			// Not accurate if there are multiple keys in this method (invalid)
			isMethod := p.CurrKind() == lexer.LeftParenthesis
			// Multiple overloads can be declared for methods
			if first, ok := fieldMap[name.Name]; ok && (!isMethod || !first.method) {
				err := klarerrs.Node(klarerrs.ErrRedeclaredField, name)
				err.SetParam("kind", "interface")
				err.Label = "Item " + klarerrs.Quote(name.Name) + " was already declared"
				hl := "It was originally declared here"
				if first.method && !isMethod {
					hl += " as a method"
				} else if !first.method && isMethod {
					hl += " as a field"
				}
				err.AddHighlight(hl, first.rang)
				p.Error(err)
			}
			fieldMap[name.Name] = declared{rang: name.Range()}
			return name
		}, 0, lexer.Comma, false)

		// Generic params (for method)
		generics := p.tryParseGenericDecl()
		if len(generics) > 0 && p.CurrKind() != lexer.LeftParenthesis {
			p.Error(klarerrs.ExpectedTokenf(
				"after generic parameters", lexer.LeftParenthesis, p.Curr(),
			))
		}

		// Method
		if p.CurrKind() == lexer.LeftParenthesis {
			// Parse function: #{ kind() -> String }
			if len(f.Keys) > 1 {
				// Invalid: x, y, z()
				p.Error(klarerrs.Slice(klarerrs.ErrIntfMultiKeyMethod, f.Keys))
			}
			fn := &ast.MethodType{
				BaseNode:      ast.BaseNode{ranges.Range{Start: p.Advance().Position}}, // (
				GenericParams: generics,
				Parameters:    p.parseMethodParams(),
			}
			if p.CurrKind() == lexer.Arrow {
				p.Advance()
				fn.ReturnType = p.ParseType(DefaultTypeBindingPower)
			}
			fn.Range.End = p.lastTokEnd()
			fieldMap[f.Keys[0].Name] = declared{rang: fn.Range, method: true}
			f.Value = fn
		} else {
			// Type
			p.Expect(lexer.Colon, noAdvance)
			f.Value = p.ParseType(DefaultTypeBindingPower)
		}

		// Invalid default value
		if c := p.Curr(); c.Kind == lexer.Equal || c.Kind == lexer.ColonEqual {
			p.Error(klarerrs.Token(klarerrs.ErrIntfDefaultValue, p.Advance()))
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
				p.Error(klarerrs.Token(klarerrs.ErrSelfNameDiscard, p.Curr()))
			}
			f.SelfName = new(p.ParseIdentOrDiscard())
			p.Expect(lexer.Colon)
		}
		f.SelfType = new(p.ParseIdentifier()) // TODO: change type of f.Struct to allow types
		p.Expect(lexer.RightParenthesis)      // )
		p.Expect(lexer.Dot, noAdvance, expectErrorCode(klarerrs.ErrFuncDotAfterSelf))
		f.Identifier = p.ParseMapIdentifier(0)
	} else if p.PeekKind() == lexer.Dot {
		// Method declaration
		// 	func Type.method()
		f.SelfType = new(p.ParseIdentifier())
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
	f.GenericParams = p.tryParseGenericDecl()

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

func (p *Parser) tryParseGenericDecl() (generics []ast.Identifier) {
	if p.CurrKind() != lexer.LessThan {
		return
	}
	p.Advance()
	declared := map[string]ranges.Range{}
	parseSeries(p, &generics, func() ast.Identifier {
		ident := p.ParseIdentifier()
		name := ident.Name
		if r, ok := declared[name]; ok {
			err := klarerrs.Range(klarerrs.ErrRedeclaredGeneric, ident.Range())
			err.AddHighlight("It was originally declared here", r)
			err.Label = klarerrs.Quote(name) + " already exists"
			err.Name = name
			p.Error(err)
		} else {
			declared[name] = ident.Range()
		}
		return ident
	}, lexer.GreaterThan, lexer.Comma, false)
	if len(generics) == 0 {
		p.Error(klarerrs.Token(klarerrs.ErrEmptyGeneric, p.PeekBehind()))
	}
	return
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
	p.Expect(lexer.Colon, noAdvance)
	if !isAssignment(p.PeekBehind().Kind) {
		param.Type = p.ParseType(DefaultTypeBindingPower)
	}

	// Default value:
	// 	func List.join(by by: String = ", ")
	if p.isEqual(p.Curr()) {
		if len(param.Names) > 1 {
			err := klarerrs.Range(klarerrs.ErrChainedDefault, ranges.Range{
				Start: param.Names[len(param.Names)-1].Range.Start,
				End:   param.Type.GetRange().End,
			})
			err.Highlights = append(err.Highlights, klarerrs.Highlight{
				Range: ranges.Between(
					param.Names[0].Range,
					param.Names[len(param.Names)-2].Range,
				),
			})
			p.ErrorLabelled(err, "Separate these parameters")
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
		p.Error(klarerrs.Node(klarerrs.ErrGenericInFuncAlias, f.Identifier))
	}
	if f.SelfName != nil {
		p.Error(klarerrs.Node(klarerrs.ErrSelfLabelInFuncAlias, f.Identifier))
	}
	if f.SelfType != nil {
		p.Expect(lexer.Dot) // TODO: better error message
	}
	target := p.ParseExpression(ExpressionBindingPower)
	switch target := target.(type) {
	case *ast.Symbol:
	case *ast.IndexExpression:
		if target.Computed {
			p.Error(klarerrs.Node(klarerrs.ErrComputedFuncAlias, target))
		}
		// LHS checked at analysis-time
	default:
		err := klarerrs.Node(klarerrs.ErrNonNameFuncAlias, target)
		err.HintWithDiff(
			"Or, did you mean to define a new function? Add parentheses after the function name.",
			&klarerrs.Diff{Edits: []klarerrs.DiffEdit{klarerrs.AddedString{
				Position: p.Tokens[beforeEqual].End(),
				String:   "()",
			}}},
		)
		p.Error(err)
	}
	return &ast.FuncAliasDeclaration{
		Identifier: f.Identifier,
		Struct:     f.SelfType,
		Target:     target,
	}
}

func (p *Parser) ParseAttribute() *ast.Attribute {
	p.Expect(lexer.At)
	d := &ast.Attribute{}
	p.flags |= isAttribute
	defer func() { p.flags &^= isAttribute }()
	d.Name = p.ParseIdentifier()
	if p.CurrKind() == lexer.LeftParenthesis {
		call := p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis))
		d.Args = call.Args
	}
	return d
}

func (p *Parser) validatePublic() {
	if p.CurrKind() == lexer.Public {
		p.Error(klarerrs.Token(klarerrs.ErrPublicGoesFirst, p.Curr()))
	}
}

func (p *Parser) ParsePublicModifier() ast.Statement {
	firstPublic := p.Expect(lexer.Public)
	var stmt ast.Statement
	switch curr := p.Curr(); curr.Kind {
	case lexer.Public:
		err := klarerrs.Token(klarerrs.ErrDuplicateModifier, curr)
		err.SetParam("modifier", lexer.Public)
		// Show where it was already used
		err.Highlights = append(err.Highlights, klarerrs.Highlight{
			ranges.FromToken(firstPublic), "It was already used here",
		})
		p.Error(err)
		// Still parse it
		stmt = &ast.BadExpression{Value: p.ParsePublicModifier()}
		markStartEndPos(p, stmt, curr.Position)
	default:
		stmt = p.ParseStatement(noEOS)
	}
	switch stmt.(type) {
	case *ast.BadExpression:
	case *ast.FunctionDeclaration, ast.TypeDeclaration,
		*ast.VariableDeclaration, *ast.FuncAliasDeclaration,
		ast.ModifierDeclaration:
		return &ast.PublicDeclaration{Declaration: stmt}
	default:
		p.Error(klarerrs.Node(klarerrs.ErrInvalidPublic, stmt))
	}
	return &ast.BadExpression{Value: stmt}
}
