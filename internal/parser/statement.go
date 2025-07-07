package parser

import (
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseVarTypeAnnotation(left ast.Node, bp BindingPower) *ast.TypeAnnotation {
	// LHS must be a Symbol or index
	if _, ok := left.(ast.Assignable); !ok {
		p.Error(errors.ParseError{
			ErrorCode: errors.ErrExpectedSymbolAssign,
			Node:      left,
		})
	}
	// Skip the :
	p.Advance()
	typ := p.ParseType(bp)
	if !p.isWhenCase && p.CurrentTokenKind() != lexer.ColonEqual {
		p.Error(errors.ExpectedToken(lexer.ColonEqual, p.CurrentToken()))
	}
	return &ast.TypeAnnotation{
		Variable: left.(ast.Expression),
		Type:     typ,
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(left ast.Expression, bp BindingPower) ast.Statement {
	op := p.Advance().Kind

	rhs := p.ParseExpression(bp)
	if op == lexer.ColonEqual {
		var explicitType ast.Type
		switch annot := left.(type) {
		case ast.Assignable:
		case *ast.TypeAnnotation:
			explicitType = annot.Type
			left = annot.Variable
		default:
			left = &ast.BadExpression{Value: left}
		}
		// Constants are ALL_CAPS
		// Limitation: if the name is written in a script without distinct
		// capital letters, we can't tell if it is all caps or not, so it
		// is just not constant.
		
		var isConst bool
		if symbol, ok := left.(*ast.Symbol); ok {
			id := symbol.Identifier
			upper := strings.ToUpper(id)
			isConst = id == upper && upper != strings.ToLower(id)
		}
		return &ast.VariableDeclaration{
			Assignee:     left,
			Constant:     isConst,
			Value:        rhs,
			ExplicitType: explicitType,
		}
	}
	if !p.validateAssignable(left) {
		left = &ast.BadExpression{Value: left}
	}
	return &ast.AssignmentStatement{
		Assignee: left.(ast.Assignable),
		Operator: op,
		Value:    rhs,
	}
}

func (p *Parser) ParseImportStatement() *ast.ImportStatement {
	var (
		module, alias string
		unqualImports []*ast.UnqualifiedImport
		isWildcard    bool
	)
	// Skip import keyword
	p.Expect(lexer.Import)

	// Parse maybe alias
	module = p.Expect(lexer.Identifier).Source
	if p.CurrentTokenKind() == lexer.Equal {
		alias, module = module, alias
		p.Advance()
	}

	for p.HasTokens() && p.IsCurrently(lexer.Identifier, lexer.Dot) {
		module += p.Advance().Source
	}
	if p.CurrentTokenKind() == lexer.Asterisk {
		// Wildcard
		module += "*"
		isWildcard = true
		p.Tokens = slices.Insert(p.Tokens, p.Index+1, lexer.Token{
			Kind:   lexer.EndOfStatement,
			Source: "\n",
			Position: lexer.Position{
				Line: p.CurrentToken().Position.Line,
				Col:  p.CurrentToken().Position.Col + 1,
			},
		})
		p.Advance()
	}
	// Module name begins with .
	if module[0] == '.' {
		p.Error(errors.ParseError{
			ErrorCode: errors.ErrImportPrefixDot,
			Params:    map[string]any{"module": module},
		})
		module = module[1:]
	}

	// Unqualified import
	if !isWildcard && p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		p.Expect(lexer.LeftCurlyBrace)
		// import module{...} instead of module.{...}
		if module[len(module)-1] != '.' {
			p.Error(errors.Token(
				errors.ErrExpectedDotInBraceImport, p.PeekBehind(),
			))
		} else {
			module = module[:len(module)-1]
		}
		// Empty import
		if p.CurrentTokenKind() == lexer.RightBracket {
			p.Error(errors.Token(errors.ErrEmptyUnqImport, p.CurrentToken()))
		}

		var (
			wasTypeKw, isTypeImport bool
			alias                   string
		)
		for p.WhileNotEndOr(lexer.RightCurlyBrace) {
			if wasTypeKw && !p.IsCurrently(lexer.Identifier, lexer.Asterisk) {
				p.Error(errors.ExpectedToken(lexer.Identifier, p.CurrentToken()))
			}
			wasTypeKw = false
			switch p.CurrentTokenKind() {
			case lexer.Type:
				isTypeImport, wasTypeKw = true, true
				p.Advance()
				continue
			case lexer.Identifier:
				// Alias:
				// 	.{fetch: get}
				// 	.{Reader: type BufferedReader}
				// Wildcard not allowed (alias: type *)
				if alias == "" && p.Peek().Kind == lexer.Colon {
					name := p.Advance().Source
					p.Advance() // :
					alias = name
					continue
				}
				unqualImports = append(unqualImports, &ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Alias:      alias,
					Identifier: p.CurrentToken().Source,
				})
				alias = ""
			case lexer.Asterisk:
				if alias != "" {
					p.Error(errors.Token(errors.ErrWildcardAndAlias, p.CurrentToken()))
				}
				unqualImports = append(unqualImports, &ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Wildcard:   true,
				})
			default:
				// Need identifier
				p.Error(errors.ExpectedToken(
					lexer.Identifier,
					p.CurrentToken(),
				))
			}
			p.Advance() // Move to comma or }
			if p.CurrentTokenKind() == lexer.EndOfStatement {
				p.Advance()
				continue
			}
			if p.IsNotCurrentlyEndOr(lexer.RightCurlyBrace) {
				p.Expect(lexer.Comma)
			}
		}
		// Check for invalid .{a:} or .{type}
		if wasTypeKw || alias != "" {
			p.Error(errors.ExpectedToken(lexer.Identifier, p.CurrentToken()))
		}
		p.Expect(lexer.RightCurlyBrace)
	}

	return &ast.ImportStatement{
		UnqualifiedImports: unqualImports,
		Alias:              alias,
		Module:             module,
		Wildcard:           isWildcard,
	}
}

func (p *Parser) ParseReturnStatement() *ast.ReturnStatement {
	p.Expect(lexer.Return)
	if p.CurrentTokenKind() == lexer.EndOfStatement {
		return &ast.ReturnStatement{}
	}
	return &ast.ReturnStatement{
		Value: p.ParseExpression(DefaultBindingPower),
	}
}

func (p *Parser) ParsePostfix(left ast.Expression) *ast.UpdateStatement {
	op := p.Expect(lexer.PlusPlus, lexer.MinusMinus).Kind
	return &ast.UpdateStatement{Operator: op, Left: left}
}

func (p *Parser) ParseForStatement() *ast.ForStatement {
	p.Expect(lexer.For)
	f := &ast.ForStatement{}
	next := p.Peek().Kind
	switch {
	case p.CurrentTokenKind() == lexer.LeftCurlyBrace:
		// for { - infinite loop
		f.Infinite = true
	case next == lexer.In, next == lexer.Comma:
		// for-in
		parseSeries(p, &f.Variables, func() string {
			return p.Expect(lexer.Identifier, lexer.Underscore).Source
		}, lexer.In, lexer.Comma, false)
		fallthrough
	default:
		f.Expression = p.ParseExpression(ExpressionBindingPower)
	}
	f.Body = p.ParseBlock()
	return f
}

func (p *Parser) ParseBlock() (body []ast.Statement) {
	p.Expect(lexer.LeftCurlyBrace)
	for p.WhileNot(lexer.RightCurlyBrace) {
		body = append(body, p.ParseStatement())
	}
	p.Expect(lexer.RightCurlyBrace)
	return
}
