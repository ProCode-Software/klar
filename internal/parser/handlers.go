package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

var IsHandledNUD = []lexer.TokenType{
	lexer.Identifier, lexer.String, lexer.Numeric, lexer.Boolean, lexer.Nil,
	lexer.Minus, lexer.Plus, lexer.Not,
	lexer.LeftParenthesis, lexer.HashLeftCurlyBrace, lexer.LeftBracket,
	lexer.Dot, lexer.Ellipsis, lexer.When, lexer.Slash,
}

func (p *Parser) handleNUD(kind lexer.TokenType) (res ast.Node, handled bool) {
	startPos := p.CurrentToken().Position
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
		res = p.ParseParenExpression()
	case lexer.HashLeftCurlyBrace:
		res = p.ParseMap()
	case lexer.LeftBracket:
		if p.isListCast() {
			res = p.ParseListCast()
		} else {
			res = p.ParseList()
		}
	case lexer.Dot:
		res = p.ParseEnumLiteral()
	case lexer.Ellipsis:
		res = p.ParseLeftRest()
	case lexer.Slash:
		res = p.ParseRegexLiteral()
	case lexer.When:
		if p.isWhenGuard {
			p.Error(errors.Token(errors.ErrNotAllowedInGuard, p.CurrentToken()))
			return &ast.BadExpression{Token: kind}, true
		}
		res = p.ParseWhenBlock()
	case lexer.Underscore:
		if !p.isWhenCase {
			return nil, false
		}
		res = &ast.Discard{}
		p.Advance()
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

func (p *Parser) handleLED(
	kind lexer.TokenType, left ast.Node, bp BindingPower,
) (res ast.Node, handled bool) {
	switch kind {
	default:
		return left, false

	// Version (v1-dev)
	case lexer.Minus:
		if _, ok := left.(*ast.Symbol); ok && p.isAttribute {
			res = p.ParseVersion(left, bp)
			break
		}
		fallthrough
	case
		// Arithmetic
		lexer.Plus, lexer.Asterisk, lexer.Slash, lexer.Percent, lexer.Caret,
		// Relational
		lexer.GreaterThan, lexer.LessThan, lexer.GreaterEqualTo, lexer.LessEqualTo,
		lexer.EqualEqual, lexer.NotEqual, lexer.In,
		// Logical
		lexer.AndAnd, lexer.OrOr,
		// Distributive
		lexer.And, lexer.Or:
		res = p.ParseBinaryExpression(left, bp)
	// Type annotation
	case lexer.Colon:
		res = p.ParseVarTypeAnnotation(left, bp)
	// Index
	case lexer.Dot, lexer.LeftBracket:
		res = p.ParseIndexExpression(left, bp)
	// Call
	case lexer.LeftParenthesis:
		res = p.ParseCallExpression(left, bp)
	// Assignment
	case lexer.PlusEqual, lexer.MinusEqual, lexer.ColonEqual, lexer.Equal:
		res = p.ParseAssignment(left.(ast.Expression), bp)
	// Increment/decrement (statements, not expressions)
	case lexer.PlusPlus, lexer.MinusMinus:
		p.validateAssignable(left)
		res = p.ParsePostfix(left.(ast.Expression))
	// Arrow function
	case lexer.Arrow:
		res = p.ParseLambda(left, bp)
	// Spread or range
	case lexer.Ellipsis:
		res = p.ParseRange(left, bp)
	// Function pipeline
	case lexer.Pipeline:
		res = p.ParsePipeline(left, bp)
	case lexer.StrokeDot:
		res = p.ParseObjectPipeline(left, bp)
	// Version
	case lexer.Numeric:
		if _, ok := left.(*ast.Symbol); !ok || !p.isAttribute {
			return left, false
		}
		res = p.ParseVersion(left, bp)
	}
	res.SetPos(left.GetRange().Start, p.lastTokEnd())
	return res, true
}

func (p *Parser) validateAssignable(left ast.Node) bool {
	if _, ok := left.(ast.Assignable); !ok {
		p.Error(errors.ParseError{
			ErrorCode: errors.ErrInvalidAssignment,
			Node:      left,
		})
		return false
	}
	return true
}

// handleStatement covers all keywords
func (p *Parser) handleStatement(kind lexer.TokenType, isTopLevel bool) (res ast.Statement, handled bool) {
	startPos := p.CurrentToken().Position
	switch kind {
	default:
		if !isTopLevel {
			return nil, false
		}
		switch kind {
		default:
			return nil, false
		case lexer.Import:
			res = p.ParseImportStatement()
		case lexer.Public:
			res = p.ParsePublicModifier()
		case lexer.At:
			res = p.ParseAttribute()
		}
	case lexer.Type:
		res = p.ParseTypeDeclaration()
	case lexer.Func:
		res = p.ParseFuncDeclaration()
	case lexer.Return:
		res = p.ParseReturnStatement()
	case lexer.For:
		res = p.ParseForStatement()
	case lexer.Next:
		res = &ast.NextStatement{}
		p.Advance()
	case lexer.Break:
		res = &ast.BreakStatement{}
		p.Advance()
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

// =================
// TYPES
// =================

func (p *Parser) handleTypeNUD(kind lexer.TokenType) (res ast.Type, handled bool) {
	startPos := p.CurrentToken().Position
	switch kind {
	case lexer.LeftBracket:
		res = p.ParseListType()
	case lexer.Identifier:
		res = p.ParseTypeAlias()
	case lexer.LeftParenthesis:
		res = p.ParseTupleType()
	case lexer.Ellipsis:
		res = p.ParseRestType()
	default:
		return nil, false
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

func (p *Parser) handleTypeLED(kind lexer.TokenType, left ast.Type, bp BindingPower) (res ast.Type, handled bool) {
	switch kind {
	case lexer.Stroke:
		res = p.ParseUnionType(left, bp)
	case lexer.Question:
		res = p.ParseOptionalType(left, bp)
	case lexer.LessThan:
		res = p.ParseGenericType(left, bp)
	case lexer.Arrow:
		res = p.ParseFunctionType(left, bp)
	case lexer.Dot:
		if left, ok := left.(*ast.TypeAlias); !ok {
			return nil, false
		} else {
			res = p.ParseTypeNamespace(left, bp)
		}
	default:
		return left, false
	}
	res.SetPos(left.GetRange().Start, p.lastTokEnd())
	return res, true
}

func (p *Parser) isListCast() bool {
	i := p.Index
	brackCount := 0
loop:
	for ; ; i++ {
		tok := p.Tokens[i]
		switch tok.Kind {
		case lexer.RightBracket:
			brackCount--
			if brackCount == 0 {
				break loop
			}
		case lexer.LeftBracket:
			brackCount++
		case lexer.Stroke, lexer.Question:
			return true
		default:
			return false
		case lexer.LeftParenthesis, lexer.RightParenthesis,
			lexer.GreaterThan, lexer.LessThan, lexer.Identifier,
			lexer.Dot, lexer.Arrow, lexer.Ellipsis, lexer.Comma:
		}
	}
	return p.Tokens[i+1].Kind == lexer.LeftParenthesis
}
