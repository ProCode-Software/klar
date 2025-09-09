package parser

import (
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseVarTypeAnnotation(left *ast.DestructureVars, bp BindingPower) ast.Statement {
	p.Advance() // :
	annot := &ast.TypeAnnotation{
		Variable: left,
		Type:     p.ParseType(DefaultTypeBindingPower),
	}
	switch curr := p.Curr(); curr.Kind {
	case lexer.Equal, lexer.PlusEqual, lexer.MinusEqual:
		p.Error(errors.Node(errors.ErrInvalidTypeAnnotation, annot))
		fallthrough
	case lexer.ColonEqual:
		return p.ParseAssignment(left, bpOf(curr.Kind))
	default:
		p.Error(errors.ExpectedToken(lexer.ColonEqual, curr))
		return &ast.BadExpression{Value: annot}
	}
}

func (p *Parser) ParseVariableDeclaration(left, right ast.Expression) *ast.VariableDeclaration {
	var explicitType ast.Type
	var vars []ast.Destructure
	if annot, ok := left.(*ast.TypeAnnotation); ok {
		left, explicitType = annot.Variable, annot.Type
	}
	if left2, ok := left.(*ast.DestructureVars); ok {
		vars = make([]ast.Destructure, len(left2.Values))
		for i, v := range left2.Values {
			if _, ok := v.(ast.Destructure); !ok {
				p.Error(errors.Node(errors.ErrNonNameDeclaration, v))
				v = &ast.BadExpression{Value: v}
			}
			vars[i] = v.(ast.Destructure)
		}
	} else {
		println("!!!")
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
	}
	return &ast.VariableDeclaration{
		Variables:    vars,
		Value:        right,
		ExplicitType: explicitType,
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(left ast.Expression, bp BindingPower) ast.Statement {
	op := p.Advance()
	rhs := p.ParseExpression(bp)
	if op.Kind == lexer.ColonEqual {
		return p.ParseVariableDeclaration(left, rhs)
	}
	var values []ast.Assignable
	if l, ok := left.(*ast.DestructureVars); ok {
		values = l.Values
	} else {
		// I think this is already prevented
		panic("left side of assignment is not *ast.DestructureVars")
		// values = []ast.Assignable{&ast.BadExpression{Value: left}}
	}
	return &ast.AssignmentStatement{
		Assignee: values,
		Operator: newOperator(op),
		Value:    rhs,
	}
}

// TODO: fix
// Soft keywords are not allowed in module names
func (p *Parser) ParseImportStatement() *ast.ImportStatement {
	var (
		b             strings.Builder
		module, alias ast.Identifier
		unqualImports []*ast.UnqualifiedImport
		isWildcard    bool
		modStart      lexer.Position
	)
	// Skip import keyword
	p.Expect(lexer.Import)

	// Parse maybe alias
	if p.Peek().Kind == lexer.Equal {
		alias = p.ParseIdentifier()
		p.Advance() // =
	}

	// First part of module
	first := p.Expect(lexer.Identifier)
	b.WriteString(first.Source)
	modStart = first.Position

	for p.WhileNot(lexer.EndOfStatement) {
		if p.CurrKind() == lexer.Dot {
			p.Advance()
			if p.CurrKind() == lexer.LeftCurlyBrace {
				break
			}
			b.WriteByte('*')
			if p.CurrKind() == lexer.Asterisk {
				break
			}
		}
		b.WriteString(p.Expect(lexer.Identifier).Source)
	}
	module.SetPos(modStart, p.lastTokEnd())

	// Wildcard
	if p.CurrKind() == lexer.Asterisk {
		isWildcard = true
		p.Tokens = slices.Insert(p.Tokens, p.Index+1, lexer.Token{
			Kind:   lexer.EndOfStatement,
			Source: "\n",
			Position: lexer.Position{
				Line: p.Curr().Position.Line,
				Col:  p.Curr().Position.Col + 1,
			},
		})
		p.Advance()
	}
	module.Name = b.String()

	// Unqualified import
	if p.CurrKind() == lexer.LeftCurlyBrace {
		if isWildcard {
			// Wildcard and unqualified import
			// import module.*.{...}
			// TODO: handle
		}
		p.Expect(lexer.LeftCurlyBrace)
		// Empty import
		if p.CurrKind() == lexer.RightCurlyBrace {
			p.Error(errors.Token(errors.ErrEmptyUnqImport, p.Curr()))
		}

		// Alias and unqualified import
		if alias.Name != "" {
			p.Error(errors.Token(
				errors.ErrAliasInUnqualifiedImport, p.PeekBehind(),
			))
		}
		module.Name = module.Name[:len(module.Name)-1]

		var (
			wasTypeKw, isTypeImport bool
			alias                   ast.Identifier
		)
		for p.WhileNotEndOr(lexer.RightCurlyBrace) {
			if wasTypeKw && !p.IsCurrently(lexer.Identifier, lexer.Asterisk) {
				p.Error(errors.ExpectedToken(lexer.Identifier, p.Curr()))
			}
			wasTypeKw = false
			curr := p.CurrKind()
			switch {
			case curr == lexer.Type:
				isTypeImport, wasTypeKw = true, true
				p.Advance()
				continue
			case curr == lexer.Asterisk:
				if alias.Name != "" {
					p.Error(errors.Token(errors.ErrWildcardAndAlias, p.Curr()))
				}
				unqualImports = append(unqualImports, &ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Wildcard:   true,
				})
			case isValidIdentifier(curr):
				// Alias:
				// 	.{fetch: get}
				// 	.{Reader: type BufferedReader}
				// Wildcard not allowed (alias: type *)
				if alias.Name == "" && p.Peek().Kind == lexer.Colon {
					name := p.ParseValidIdent()
					p.Advance() // :
					alias = name
					continue
				}
				unqualImports = append(unqualImports, &ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Alias:      alias,
					Identifier: p.ParseValidIdent(),
				})
				alias.Name = ""
			default:
				// Need identifier
				p.Error(errors.ExpectedToken(lexer.Identifier, p.Curr()))
			}
			p.Advance() // Move to comma or }
			if p.IsNotCurrentlyEndOr(lexer.RightCurlyBrace) {
				p.Expect(lexer.Comma, lexer.EndOfStatement)
			}
		}
		// Check for invalid .{a:} or .{type}
		if wasTypeKw || alias.Name != "" {
			p.Error(errors.ExpectedToken(lexer.Identifier, p.Curr()))
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
	if p.CurrKind() == lexer.EndOfStatement {
		return &ast.ReturnStatement{}
	}
	return &ast.ReturnStatement{
		Value: p.ParseExpression(DefaultBindingPower),
	}
}

func (p *Parser) ParseUpdateStatement(left ast.Expression) *ast.UpdateStatement {
	op := p.Expect(lexer.PlusPlus, lexer.MinusMinus)
	return &ast.UpdateStatement{Operator: newOperator(op), Left: left}
}

func (p *Parser) ParseForStatement() *ast.ForStatement {
	p.Expect(lexer.For)
	f := &ast.ForStatement{}
	// Peek for `in` before parsing destructure
	if p.Lookahead(isDestructureAssignment) {
		f.Variables = p.ParseDestructureTypePairs(false)
		p.Expect(lexer.In)
	}
	f.Expression = p.ParseExpression(ExpressionBindingPower)
	f.Body = p.ParseBlock()
	return f
}

func (p *Parser) ParseWhileStatement() *ast.WhileStatement {
	p.Advance() // while
	w := &ast.WhileStatement{}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		w.Infinite = true
	} else {
		w.Condition = p.ParseExpression(ExpressionBindingPower)
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
