package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) handleNUD(kind lexer.TokenType) (res ast.Expression, handled bool) {
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
	case lexer.Not:
		fallthrough
	case lexer.Minus:
		res = p.ParseUnaryExpression()
	// Group or tuple
	case lexer.LeftParenthesis:
		res = p.ParseParenExpression()
	case lexer.HashLeftCurlyBrace:
		res = p.ParseMap()
	case lexer.LeftBracket:
		if p.IsListCastStart() {
			res = p.ParseListCast()
		} else {
			res = p.ParseList()
		}
	case lexer.Func:
		res = p.ParseLambda()
	case lexer.Dot:
		res = p.ParseEnumLiteral()
	case lexer.Ellipsis:
		res = p.ParseLeftRest()
	case lexer.Regex:
		res = p.ParseRegexLiteral()
	case lexer.When:
		res = p.ParseWhenBlock()
	case lexer.For:
		res = p.ParseForExpression()
	case lexer.Go:
		res = p.ParseGoExpression()
	case lexer.Await:
		res = p.ParseAwaitExpression()
	case lexer.Try:
		res = p.ParseTryExpression()
	case lexer.Underscore:
		u := p.Advance()
		if !p.isWhenCase() {
			p.Error(errors.Token(errors.ErrUnderscoreValue, u))
		}
		res = &ast.Discard{}
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

func (p *Parser) handleLED(
	kind lexer.TokenType, left ast.Expression, bp BindingPower,
) (res ast.Expression, handled bool) {
	switch kind {
	default:
		return left, false

	// Version (v1-dev)
	case lexer.Minus:
		if left, ok := left.(*ast.Symbol); ok && p.isAttribute() &&
			left.Identifier[0] == 'v' {
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
	// Index
	case lexer.Dot, lexer.LeftBracket:
		res = p.ParseIndexExpression(left, bp)
	// Call
	case lexer.LeftParenthesis:
		res = p.ParseCallExpression(left, bp)
	// Spread or range
	case lexer.Ellipsis, lexer.DotDotLessThan:
		res = p.ParseRange(left, bp)
	// Function pipeline
	case lexer.Pipeline:
		res = p.ParsePipeline(left, bp)
	// Object pipeline
	case lexer.StrokeDot:
		res = p.ParseObjectPipeline(left, bp)
	// Assertion
	case lexer.NotNot:
		res = p.ParseAssertExpression(left)
	// Version
	case lexer.Numeric:
		if !p.isAttribute() {
			return left, false
		}
		if l, ok := left.(*ast.Symbol); !ok || l.Identifier[0] != 'v' {
			return left, false
		} else {
			res = p.ParseVersion(l, bp)
		}
	// Invalid assignment
	case lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual,
		lexer.AsteriskEqual, lexer.SlashEqual, lexer.PercentEqual, lexer.CaretEqual:
		err := errors.Token(errors.ErrAssignmentAsExpr, p.Advance())
		if kind == lexer.Equal {
			err.Hint("Did you mean to use '==' instead?")
		}
		p.Error(err)
		res = &ast.BadExpression{
			Token: kind,
			Value: p.ParseExpression(ExpressionBindingPower),
		}
	}
	res.SetPos(left.GetRange().Start, p.lastTokEnd())
	return res, true
}

func (p *Parser) handleStatementLED(kind lexer.TokenType, left ast.Expression) (
	res ast.Statement, handled bool,
) {
	switch kind {
	default:
		return nil, false
	// Type annotation
	case lexer.Colon:
		res = p.ParseVarTypeAnnotation([]ast.Assignable{p.validateAssignable(left)})
	// Assignment
	case lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual,
		lexer.AsteriskEqual, lexer.SlashEqual, lexer.PercentEqual, lexer.CaretEqual:
		res = p.ParseAssignment([]ast.Assignable{p.validateAssignable(left)}, nil)
	// Declaration or assignment
	case lexer.Comma:
		res = p.ParseCommaStatement(left)
	}
	res.SetPos(left.GetRange().Start, p.lastTokEnd())
	return res, true
}

func (p *Parser) validateAssignable(node ast.Node) ast.Assignable {
	n, ok := node.(ast.Assignable)
	if ok {
		return n
	}
	p.Error(&errors.ParseError{
		ErrorCode: errors.ErrInvalidAssignment,
		Range:     node.GetRange(),
		Node:      node,
	})
	return &ast.BadExpression{Value: node}
}

func (p *Parser) handleTopLevelStatement(kind lexer.TokenType) (res ast.Statement, handled bool) {
	startPos := p.Curr().Position
	switch kind {
	default:
		return nil, false
	case lexer.Import:
		if p.PeekKind() == lexer.LeftParenthesis {
			return nil, false
		}
		res = p.ParseImportStatement()
	case lexer.Public:
		res = p.ParsePublicModifier()
	case lexer.Opaque:
		res = p.ParseOpaqueModifier()
	case lexer.At:
		res = p.ParseAttribute()
	}
	res.SetPos(startPos, p.lastTokEnd())
	return res, true
}

// handleStatement covers all keywords
func (p *Parser) handleStatement(kind lexer.TokenType) (res ast.Statement, handled bool) {
	startPos := p.Curr().Position
	switch kind {
	default:
		return nil, false
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
	case lexer.Next, lexer.Stop:
		res = p.ParseControlStatement()
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
	case lexer.Identifier:
		res = p.ParseTypeAlias()
	case lexer.LeftBracket:
		res = p.ParseListType()
	case lexer.HashLeftCurlyBrace:
		res = p.ParseMapType()
	case lexer.LeftParenthesis:
		res = p.ParseTupleType()
	case lexer.Func:
		res = p.ParseFunctionType()
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

func (p *Parser) IsListCastStart() bool {
	_, ok := p.listCastTokens[p.Index]
	return ok
}
