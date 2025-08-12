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
	p.validateAssignable(left)
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
	op := p.Advance()

	rhs := p.ParseExpression(bp)
	if op.Kind == lexer.ColonEqual {
		var explicitType ast.Type
		var vars []ast.Destructure
		switch annot := left.(type) {
		case ast.Assignable:
		case *ast.TypeAnnotation:
			explicitType = annot.Type
			left = annot.Variable
		default:
			left = &ast.BadExpression{Value: left}
		}
		return &ast.VariableDeclaration{
			Variables:    vars,
			Value:        rhs,
			ExplicitType: explicitType,
		}
	}
	if !p.validateAssignable(left) {
		left = &ast.BadExpression{Value: left}
	}
	return &ast.AssignmentStatement{
		Assignee: left.(ast.Assignable),
		Operator: newOperator(op),
		Value:    rhs,
	}
}

// Constants are ALL_CAPS
// Limitation: if the name is written in a script without distinct
// capital letters, we can't tell if it is all caps or not, so it
// is just not constant.
func isConstant(id string) bool {
	upper := strings.ToUpper(id)
	return id == upper && upper != strings.ToLower(id)
}

func (p *Parser) ParseImportStatement() *ast.ImportStatement {
	var (
		b             strings.Builder
		module        = &ast.Symbol{}
		alias         *ast.Symbol
		unqualImports []*ast.UnqualifiedImport
		isWildcard    bool
		modStart      lexer.Position
	)
	// Skip import keyword
	p.Expect(lexer.Import)

	// Parse maybe alias
	if p.Peek().Kind == lexer.Equal {
		tok := p.Expect(lexer.Identifier)
		alias = rangeFromToken(&ast.Symbol{Identifier: tok.Source}, tok)
		p.Advance() // =
	}

	// First part of module
	first := p.Expect(lexer.Identifier)
	b.WriteString(first.Source)
	modStart = first.Position

	for p.WhileNot(lexer.EndOfStatement) {
		if p.CurrentTokenKind() == lexer.Dot {
			p.Advance()
			if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
				break
			}
			b.WriteByte('*')
			if p.CurrentTokenKind() == lexer.Asterisk {
				break
			}
		}
		b.WriteString(p.Expect(lexer.Identifier).Source)
	}
	module.SetPos(modStart, p.lastTokEnd())

	// Wildcard
	if p.CurrentTokenKind() == lexer.Asterisk {
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
	module.Identifier = b.String()

	// Unqualified import
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		if isWildcard {
			// Wildcard and unqualified import
			// import module.*.{...}
		}
		p.Expect(lexer.LeftCurlyBrace)
		// Empty import
		if p.CurrentTokenKind() == lexer.RightCurlyBrace {
			p.Error(errors.Token(errors.ErrEmptyUnqImport, p.CurrentToken()))
		}

		// Alias and unqualified import
		if alias != nil {
			p.Error(errors.Token(
				errors.ErrAliasInUnqualifiedImport, p.PeekBehind(),
			))
		}
		module.Identifier = module.Identifier[:len(module.Identifier)-1]

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
	op := p.Expect(lexer.PlusPlus, lexer.MinusMinus)
	return &ast.UpdateStatement{Operator: newOperator(op), Left: left}
}

func (p *Parser) ParseForStatement() *ast.ForStatement {
	p.Expect(lexer.For)
	f := &ast.ForStatement{}
	if c := p.CurrentTokenKind(); c == lexer.LeftBracket || c == lexer.LeftParenthesis {
		// Peek for `in` before parsing destructure
	}
	f.Variables = p.ParseDestructureSeries()
	p.Expect(lexer.In)
	f.Expression = p.ParseExpression(ExpressionBindingPower)
	f.Body = p.ParseBlock()
	return f
}

func (p *Parser) ParseWhileStatement() *ast.WhileStatement {
	p.Advance() // while
	w := &ast.WhileStatement{}
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		w.Infinite = true
	} else {
		w.Condition = p.ParseExpression(ExpressionBindingPower)
		p.Expect(lexer.LeftCurlyBrace)
	}
	w.Body = p.ParseBlock()
	return w
}

func (p *Parser) ParseBlock() (body []ast.Statement) {
	p.Expect(lexer.LeftCurlyBrace)
	for p.WhileNot(lexer.RightCurlyBrace) {
		body = append(body, p.ParseStatement())
	}
	p.Expect(lexer.RightCurlyBrace)
	return
}

func isAssignment(kind lexer.TokenType) bool {
	switch kind {
	case lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual:
		return true
	}
	return false
}

// Validate the += or -= operator at
func (p *Parser) ParseDestructureDeclaration() ast.Statement {
	var left ast.Expression = &ast.DestructureVars{Values: p.ParseDestructureSeries()}
	if p.CurrentTokenKind() == lexer.Colon {
		left = p.ParseVarTypeAnnotation(left, AssignBindingPower)
	}
	return p.ParseAssignment(left, AssignBindingPower)
}
