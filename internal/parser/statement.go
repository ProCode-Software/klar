package parser

import (
	"fmt"
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

func (p *Parser) ParseImportStatement() ast.ImportStatement {
	var (
		module        string
		unqualImports []ast.UnqualifiedImport
		isWildcard    bool
	)
	// Skip import keyword
	p.Expect(lexer.Import)

	for p.HasTokens() && (p.CurrentTokenKind() == lexer.Identifier ||
		p.CurrentTokenKind() == lexer.Dot ||
		p.CurrentTokenKind() == lexer.Times) {
		module += p.Advance().Source
	}

	// Module name begins with .
	if module[0] == '.' {
		panic(errors.ParseError{
			Type:   errors.ErrImportPrefixDot,
			Params: map[string]any{"module": module},
		})
	}
	wcCount := strings.Count(module, "*")
	switch {
	case wcCount < 1:
	case wcCount > 1:
		panic(errors.NewTokenError(errors.ErrImportTooManyWildcard, p.CurrentToken()))
	// import klar.*.{
	case strings.Index(module, "*.") == len(module)-2:
		panic(errors.NewTokenError(errors.ErrWildcardAndUnqImport, p.CurrentToken()))
	case strings.Index(module, ".*") != len(module)-2:
		panic(errors.NewTokenError(errors.ErrImportInvalidWildcard, p.CurrentToken()))
	default:
		isWildcard = true
		fmt.Println("WILDCARD")
		fmt.Println(p.CurrentToken())
		// EOS insertion doesn't add after *. We need to add it manually.
		p.Tokens = slices.Insert(
			p.Tokens, p.Index+1, lexer.Token{Kind: lexer.EndOfStatement},
		)
	}
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		// import .{...}
		if module == "" || module == "." {
			panic(errors.NewTokenError(
				errors.ErrImportExpectedModule, p.CurrentToken(),
			))
		}
		// import module{...} instead of module.{...}
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
				p.ExpectNext(lexer.Identifier, lexer.Times)
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
	}
	p.Advance()
	return ast.ImportStatement{
		UnqualifiedImports: unqualImports,
		Module:             module,
		Wildcard:           isWildcard,
	}
}

func (p *Parser) ParseTypeDeclaration() ast.TypeDeclaration {
	for _, tok := range p.Tokens {
		fmt.Printf("%-20s %s", lexer.TokenTypes[tok.Kind], tok.Source)
	}
	p.Expect(lexer.Type)
	name := p.Expect(lexer.Identifier)
	switch p.Advance().Kind {
	case lexer.EqualSign:
		// Type
		return ast.TypeAliasDeclaration{
			Identifier: name.Source,
			Type: p.ParseType(AssignBindingPower, false),
		}
	case lexer.LeftParenthesis:
		// Inherited struct
		panic("TODO")
	case lexer.LeftCurlyBrace:
		// Struct or enum
	default:
		// Some other token or unassigned type (if EOS)
		panic(errors.NewTokenError(errors.ErrExpectedTypeAssignment, p.CurrentToken()))
	}
	return nil
}