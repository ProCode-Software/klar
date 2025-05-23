package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) handleNUD(kind lexer.TokenType) (res ast.ASTItem, handled bool) {
	switch kind {
	default:
		return nil, false
	// Primary expression/literal
	case lexer.Identifier, lexer.String, lexer.Numeric, lexer.Boolean, lexer.Nil:
		res = p.ParsePrimaryExpression()
	// Prefix/Unary
	case lexer.Minus, lexer.Plus, lexer.Not:
		res = p.ParseUnaryExpression()
	// Group or tuple
	case lexer.LeftParenthesis:
		res = p.ParseGroupOrTuple()
	}
	return res, true
}

func (p *Parser) handleLED(
	kind lexer.TokenType, left ast.ASTItem, bp BindingPower,
) (res ast.ASTItem, handled bool) {
	switch kind {
	default:
		return nil, false

	// Arithmetic
	case lexer.Plus, lexer.Minus, lexer.Asterisk, lexer.Slash, lexer.Percent:
		res = p.ParseBinaryExpression(left, BindingPowerMap[p.CurrentTokenKind()])

	// Exponent (right-associative)
	case lexer.Caret:
		res = p.ParseBinaryExpression(left, BindingPowerMap[p.CurrentTokenKind()]-1)

	// Type annotation
	case lexer.Colon:
		res = p.ParseVarTypeAnnotation(left, BindingPowerMap[p.CurrentTokenKind()])

	// Assignment
	case lexer.PlusEqual, lexer.MinusEqual, lexer.ColonEqual, lexer.Equal:
		// Left must be an expression
		if _, ok := left.(ast.Expression); !ok {
			panic(errors.ParseError{
				Type:    errors.ErrExpectedSymbolAssign,
				ASTItem: left,
			})
		}
		res = p.ParseAssignment(left.(ast.Expression), bp)
	}
	return res, true
}

func (p *Parser) handleStatement(kind lexer.TokenType) (res ast.Statement, handled bool) {
	switch kind {
	default:
		return nil, false
	// Import
	case lexer.Import:
		res = p.ParseImportStatement()
	case lexer.Type:
		res = p.ParseTypeDeclaration()
	}
	return res, true // TODO: add statements
}

// =================
// TYPES
// =================

func (p *Parser) handleTypeNUD(kind lexer.TokenType) (res ast.Type, handled bool) {
	switch kind {
	case lexer.LeftBracket:
		res = p.ParseListType()
	case lexer.Identifier:
		res = p.ParseTypeAlias()
	case lexer.HashLeftCurlyBrace:
		res = p.ParseInterfaceType()
	case lexer.LeftParenthesis:
		res = p.ParseTupleType()
	// TODO: map and tuple/func
	default:
		return nil, false
	}
	return res, true
}

func (p *Parser) handleTypeLED(kind lexer.TokenType, left ast.Type, bp BindingPower) (res ast.Type, handled bool) {
	switch kind {
	case lexer.Plus, lexer.Stroke:
		res = p.ParseUnionType(left, bp)
	case lexer.Question:
		res = p.ParseOptionalType(left, bp)
	case lexer.LessThan:
		res = p.ParseGenericType(left, bp)
	case lexer.Arrow:
		res = p.ParseFunctionType(left, bp)
	default:
		return nil, false
	}
	return res, true
}
