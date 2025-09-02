package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) handleNUD(kind lexer.TokenType) (res ast.Node, handled bool) {
	startPos := p.Curr().Position
	switch kind {
	default:
		if isValidIdentifier(kind) {
			res = p.ParseSymbol()
			break
		}
		return nil, false
	// Primary expression/literal
	case lexer.Identifier:
		res = p.ParseSymbol()
	case lexer.String:
		res = p.ParseString()
	case lexer.Numeric:
		res = p.ParseNumber()
	case lexer.Boolean:
		res = p.ParseBoolean()
	case lexer.Nil:
		res = p.ParseNil()
	// Prefix/Unary
	case lexer.Minus, lexer.Plus, lexer.Not:
		res = p.ParseUnaryExpression()
	// Group or tuple
	case lexer.LeftParenthesis:
		res = p.ParseParenExpression()
	case lexer.HashLeftCurlyBrace:
		res = p.ParseMap()
	case lexer.LeftBracket:
		if p.Lookahead(isListCast) {
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
		if p.isWhenGuard || p.isWhenCase {
			p.Error(errors.Token(errors.ErrNotAllowedInWhen, p.Curr()))
			return &ast.BadExpression{Token: kind}, true
		}
		res = p.ParseWhenBlock()
	case lexer.For:
		res = p.ParseForExpression()
	case lexer.Go:
		res = p.ParseGoExpression()
	case lexer.Await:
		res = p.ParseAwaitExpression()
	case lexer.Underscore:
		if u := p.Advance(); !p.isWhenCase {
			p.Error(errors.Token(errors.ErrUnderscoreValue, u))
		}
		res = &ast.Discard{}
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
		lexer.In, lexer.NotIn,
		// Logical
		lexer.AndAnd, lexer.OrOr,
		// Distributive
		lexer.And, lexer.Or:
		res = p.ParseBinaryExpression(left, bp)
	// Relational
	case lexer.GreaterThan, lexer.LessThan, lexer.GreaterEqualTo, lexer.LessEqualTo,
		lexer.EqualEqual, lexer.NotEqual:
		res = p.ParseRelationalExpression(left, bp)
	// Type annotation
	case lexer.Colon:
		if left, ok := left.(*ast.DestructureVars); ok {
			res = p.ParseVarTypeAnnotation(left, bp)
		} else {
			return nil, false
		}
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
		if p.isWhenGuard || p.isWhenCase {
			p.Error(errors.Node(errors.ErrNotAllowedInWhen, res))
			return &ast.BadExpression{Value: res}, true
		}
	// Spread or range
	case lexer.Ellipsis, lexer.DotDotLessThan:
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
	startPos := p.Curr().Position
	switch kind {
	default:
		if !isTopLevel {
			return nil, false
		}
		switch kind {
		default:
			return nil, false
		case lexer.Import:
			if !p.isModifierUse(kind) {
				return nil, false
			}
			res = p.ParseImportStatement()
		case lexer.Public:
			if !p.isModifierUse(kind) {
				return nil, false
			}
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
	case lexer.While:
		res = p.ParseWhileStatement()
	case lexer.Next:
		res = &ast.NextStatement{}
		p.Advance()
	case lexer.Break:
		res = &ast.BreakStatement{}
		p.Advance()
	case lexer.Opaque:
		if !p.isModifierUse(kind) {
			return nil, false
		}
		res = p.ParseOpaqueModifier()
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

func (p *Parser) handleStatementNUD(kind lexer.TokenType) (res ast.Expression, handled bool) {
	startPos := p.Curr().Position
	switch kind {
	case lexer.LeftBracket, lexer.HashLeftCurlyBrace, lexer.LeftParenthesis,
		// For better errors
		lexer.Numeric, lexer.Boolean, lexer.Nil, lexer.Regex:
		if p.Lookahead(isDestructureAssignment) {
			res = p.ParseDestructureVars()
			break
		}
		return nil, false
	default:
		if isValidIdentOrDiscard(kind) && p.Lookahead(isDestructureAssignment) {
			res = p.ParseDestructureVars()
			break
		}
		return nil, false
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

// =================
// TYPES
// =================

func (p *Parser) handleTypeNUD(kind lexer.TokenType) (res ast.Type, handled bool) {
	startPos := p.Curr().Position
	switch kind {
	case lexer.LeftBracket:
		res = p.ParseListType()
	case lexer.Identifier:
		res = p.ParseTypeAlias()
	case lexer.LeftParenthesis:
		res = p.ParseTupleType()
		// Convert single item tuple to paren type, unless function type
		// to avoid recreating tuple.
		if tuple := res.(*ast.TupleType); tuple.Single &&
			(p.isWhenCase || p.CurrKind() != lexer.Arrow) {
			res = &ast.ParenType{BaseNode: tuple.BaseNode, Type: tuple.Values[0].Value}
		}
	case lexer.Ellipsis:
		res = p.ParseRestType()
	case lexer.Stroke:
		res = p.ParseUnionType(nil, UnionTypeBindingPower)
	default:
		if isValidIdentifier(kind) {
			res = p.ParseTypeAlias()
			break
		}
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
