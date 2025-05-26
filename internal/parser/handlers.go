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
	case lexer.HashLeftCurlyBrace:
		res = p.ParseMap()
	case lexer.LeftBracket:
		res = p.ParseList()
	}
	return res, true
}

func (p *Parser) handleLED(
	kind lexer.TokenType, left ast.ASTItem, bp BindingPower,
) (res ast.ASTItem, handled bool) {
	currentBP := BindingPowerMap[p.CurrentTokenKind()]
	switch kind {
	default:
		return nil, false

	// Arithmetic
	case lexer.Plus, lexer.Minus, lexer.Asterisk, lexer.Slash, lexer.Percent,
	lexer.GreaterThan, lexer.LessThan, lexer.GreaterEqualTo, lexer.LessEqualTo,
	lexer.EqualEqual, lexer.NotEqual:
		res = p.ParseBinaryExpression(left, currentBP)

	// Exponent (right-associative)
	case lexer.Caret:
		res = p.ParseBinaryExpression(left, currentBP-1)

	// Type annotation
	case lexer.Colon:
		res = p.ParseVarTypeAnnotation(left, currentBP)

	// Index
	case lexer.Dot, lexer.LeftBracket:
		res = p.ParseIndexExpression(left, bp)

	// Call
	case lexer.LeftParenthesis:
		res = p.ParseCallExpression(left, bp)

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

// handleStatement covers all keywords
func (p *Parser) handleStatement(kind lexer.TokenType) (res ast.Statement, handled bool) {
	switch kind {
	default:
		return nil, false
	case lexer.Import:
		res = p.ParseImportStatement()
	case lexer.Type:
		res = p.ParseTypeDeclaration()
	case lexer.Func:
		res = p.ParseFuncDeclaration()
	case lexer.Return:
		res = p.ParseReturnStatement()
	case lexer.When:
		panic("TODO")
	case lexer.For:
		panic("TODO")
	case lexer.Next:
		res = ast.NextStatement{}
		p.Advance()
	}
	return res, true
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
	case lexer.LeftParenthesis:
		res = p.ParseTupleType()
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
