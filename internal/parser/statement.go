package parser

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseVarTypeAnnotation(left ast.ASTItem, bp BindingPower) ast.TypeAnnotation {
	// LHS must be a Symbol
	if _, ok := left.(ast.Symbol); !ok {
		panic(errors.ParseError{
			Type:    errors.ErrExpectedSymbolAssign,
			ASTItem: left,
		})
	}
	// Skip the :
	p.Advance()
	typ := p.ParseType(bp, true).(ast.SimpleType)

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
		if annot, is := left.(ast.TypeAnnotation); is {
			explicitType = annot.Type
			left = annot.Variable
		} else if _, ok := left.(ast.Symbol); !ok {
			panic(errors.ParseError{
				Type:    errors.ErrExpectedSymbolAssign,
				ASTItem: left,
			})
		}
		id := left.(ast.Symbol).Identifier
		return ast.VariableDeclaration{
			Identifier:   id,
			Constant:     strings.ToUpper(id) == id, // Constants are ALL_CAPS
			Value:        rhs,
			ExplicitType: explicitType,
		}
	}
	return ast.AssignmentStatement{
		Assignee: left,
		Operator: op,
		Value:    rhs,
	}
}

func (p *Parser) ParseImportStatement() ast.ImportStatement {
	// Skip import keyword
	p.Advance()
	var module string
	for p.CurrentTokenKind() == lexer.Identifier ||
		p.CurrentTokenKind() == lexer.Dot ||
		p.CurrentTokenKind() == lexer.Times {
		module += p.CurrentToken().Source
		p.Advance()
	}
	var unqualImports []ast.UnqualifiedImport
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		if module == "" || module == "." {
			panic(errors.NewTokenError(
				errors.ErrExpectedModuleInImport, p.CurrentToken(),
			))
		}
		if module[len(module)-1] != '.' {
			panic(errors.NewTokenError(
				errors.ErrExpectedDotInBraceImport, p.CurrentToken(),
			))
		}
		module = module[:len(module)-1]
		// Unqualified Import
		isTypeImport := false
		for p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Advance() // Skip curly brace or comma
			switch p.CurrentTokenKind() {
			case lexer.RightCurlyBrace:
				break // Traling comma
			case lexer.Type:
				isTypeImport = true
				p.Advance()
			case lexer.Identifier:
				unqualImports = append(unqualImports, ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Identifier: p.CurrentToken().Source,
				})
			case lexer.Times:
				unqualImports = append(unqualImports, ast.UnqualifiedImport{
					TypeImport: isTypeImport,
					Wildcard:   true,
				})
			default:
				// Need identifier
				panic(errors.ExpectedTokenError(
					lexer.Identifier,
					p.CurrentToken(),
					p.CurrentToken().Position,
				))
			}
			p.Advance() // Move to comma or }
			if p.CurrentTokenKind() == lexer.Comma {
				continue
			}
		}
		p.Expect(lexer.RightCurlyBrace)
		p.Advance()
	}
	return ast.ImportStatement{
		UnqualifiedImports: unqualImports,
		Module:             module,
	}
}
