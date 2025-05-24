package parser

import (
	"slices"
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
		panic(errors.ParseError{
			Type:   errors.ErrImportPrefixDot,
			Params: map[string]any{"module": module},
		})
	}

	// Unqualified import
	if !isWildcard && p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		p.Expect(lexer.LeftCurlyBrace)
		// import module{...} instead of module.{...}
		if module[len(module)-1] != '.' {
			panic(errors.NewTokenError(
				errors.ErrExpectedDotInBraceImport, p.CurrentToken(),
			))
		}
		module = module[:len(module)-1]

		var wasTypeKw, isTypeImport bool
		for p.IsNot(lexer.RightCurlyBrace) {
			if wasTypeKw && !p.IsCurrently(lexer.Identifier, lexer.Asterisk) {
				panic(errors.NewTokenError(
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
				panic(errors.ExpectedTokenError(
					lexer.Identifier,
					p.CurrentToken(),
					p.CurrentToken().Position,
				))
			}
			p.Advance() // Move to comma or }
			if !wasTypeKw && p.CurrentTokenKind() != lexer.RightCurlyBrace {
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
