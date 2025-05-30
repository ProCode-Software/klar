package parser

import (
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseVarTypeAnnotation(left ast.Node, bp BindingPower) ast.TypeAnnotation {
	// LHS must be a Symbol or index
	if _, ok := left.(ast.Assignable); !ok {
		p.Error(errors.ParseError{
			Type: errors.ErrExpectedSymbolAssign,
			Node: left,
		})
	}
	// Skip the :
	p.Advance()
	typ := p.ParseType(bp)
	if p.CurrentTokenKind() != lexer.ColonEqual {
		p.Error(errors.ExpectedToken(lexer.ColonEqual, p.CurrentToken()))
	}
	return ast.TypeAnnotation{
		Variable: left.(ast.Symbol),
		Type:     typ,
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(left ast.Expression, bp BindingPower) ast.Statement {
	op := p.Advance().Kind

	rhs := p.ParseExpression(bp)
	if op == lexer.ColonEqual {
		var explicitType ast.SimpleType
		switch annot := left.(type) {
		case ast.Assignable:
		case ast.TypeAnnotation:
			explicitType = annot.Type
			left = annot.Variable
		default:
			left = ast.BadExpression{}
		}
		id := left.(ast.Symbol).Identifier
		return ast.VariableDeclaration{
			Identifier:   id,
			Constant:     strings.ToUpper(id) == id, // Constants are ALL_CAPS
			Value:        rhs,
			ExplicitType: explicitType,
		}
	}
	if !p.validateAssignable(left) {
		left = ast.BadExpression{}
	}
	return ast.AssignmentStatement{
		Assignee: left.(ast.Assignable),
		Operator: op,
		Value:    rhs,
	}
}

// TODO: unqualified aliases
func (p *Parser) ParseImportStatement() ast.ImportStatement {
	var (
		module, alias string
		unqualImports []ast.UnqualifiedImport
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
			Type:   errors.ErrImportPrefixDot,
			Params: map[string]any{"module": module},
		})
		// module = module[1:]
	}

	// Unqualified import
	if !isWildcard && p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		p.Expect(lexer.LeftCurlyBrace)
		// import module{...} instead of module.{...}
		if module[len(module)-1] != '.' {
			p.Error(errors.Token(
				errors.ErrExpectedDotInBraceImport, p.CurrentToken(),
			))
		}
		module = module[:len(module)-1]

		var wasTypeKw, isTypeImport bool
		for p.WhileNotEndOr(lexer.RightCurlyBrace) {
			if wasTypeKw && !p.IsCurrently(lexer.Identifier, lexer.Asterisk) {
				p.Error(errors.Token(
					errors.ErrImportExpectedIdentAfterType, p.CurrentToken(),
				))
			}
			wasTypeKw = false
			switch p.CurrentTokenKind() {
			case lexer.Type:
				isTypeImport, wasTypeKw = true, true
			case lexer.Identifier:
				unqualImports = append(unqualImports, ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Identifier: p.CurrentToken().Source,
				})
			case lexer.Asterisk:
				unqualImports = append(unqualImports, ast.UnqualifiedImport{
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
			if !wasTypeKw && p.IsNotCurrentlyEndOr(lexer.RightCurlyBrace) {
				p.Expect(lexer.Comma)
			}
		}
		p.Expect(lexer.RightCurlyBrace)
	}
	return ast.ImportStatement{
		UnqualifiedImports: unqualImports,
		Alias:              alias,
		Module:             module,
		Wildcard:           isWildcard,
	}
}

func (p *Parser) ParseReturnStatement() ast.ReturnStatement {
	p.Expect(lexer.Return)
	return ast.ReturnStatement{
		Value: p.ParseExpression(DefaultBindingPower),
	}
}

func (p *Parser) ParsePostfix(left ast.Expression) ast.UpdateStatement {
	op := p.Expect(lexer.PlusPlus, lexer.MinusMinus).Kind
	return ast.UpdateStatement{Operator: op, Left: left}
}

func (p *Parser) ParseForStatement() ast.ForStatement {
	p.Expect(lexer.For)
	f := ast.ForStatement{}
	// for { - infinite loop
	if p.CurrentTokenKind() != lexer.LeftParenthesis {
		switch stmt := p.ParseStatement().(type) {
		// Conditional: for i < 10
		case ast.ExpressionStatement:
			f.Condition = stmt.Expression
		// Iteration: for i := 1...10
		case ast.AssignmentStatement:
			// TODO: comma assignments
			if _, ok := stmt.Assignee.(ast.Symbol); !ok ||
				stmt.Operator != lexer.ColonEqual {
				p.Error(errors.Node(errors.ErrForInvalidCondition, stmt))
			}
			f.Variables = append(f.Variables, stmt.Assignee.(ast.Symbol))
			f.Assignment = stmt.Value
		default:
			p.Error(errors.Node(errors.ErrForInvalidCondition, stmt))
		}
	} else {
		f.Infinite = true
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
