package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func (p *Parser) ParseBinaryExpression(left ast.Node, bp BindingPower) ast.BinaryExpression {
	op := p.Advance()
	right := p.ParseExpression(bp)
	return ast.BinaryExpression{
		Left:     left,
		Operator: op.Kind,
		Right:    right,
	}
}

func (p *Parser) ParseUnaryExpression() ast.UnaryExpression {
	op := p.Advance().Kind
	right := p.ParseExpression(UnaryBindingPower)
	return ast.UnaryExpression{Operator: op, Right: right}
}

func (p *Parser) ParseGroupOrTuple() ast.Expression {
	p.Advance() // (
	if p.CurrentTokenKind() == lexer.RightParenthesis {
		// Empty tuple
		p.Advance()
		return ast.TupleLiteral{}
	}
	expr := p.ParseExpression(CommaBindingPower)
	next := p.CurrentToken()
	switch next.Kind {
	case lexer.Comma:
		// Tuple (requires at least one comma)
		tuple := ast.TupleLiteral{}
		tuple.Values = append(tuple.Values, expr)
		p.Advance()
		for p.WhileNotEndOr(lexer.RightParenthesis) {
			tuple.Values = append(tuple.Values, p.ParseExpression(LogicalBindingPower))
			if p.CurrentTokenKind() != lexer.RightParenthesis {
				p.Expect(lexer.Comma)
			}
		}
		p.Expect(lexer.RightParenthesis)
		return tuple
	case lexer.RightParenthesis:
		// Grouped expression
		p.Advance()
		return expr
	default:
		panic(errors.ExpectedTokenError(lexer.RightParenthesis, next))
	}
}

func (p *Parser) ParseMap() ast.MapLiteral {
	p.Expect(lexer.HashLeftCurlyBrace)
	entries := []ast.Pair{}
	for p.WhileNotEndOr(lexer.RightCurlyBrace) {
		entry := ast.Pair{
			Key: p.ParseExpression(LogicalBindingPower),
		}
		p.Expect(lexer.Colon)
		entry.Value = p.ParseExpression(LogicalBindingPower)
		entries = append(entries, entry)
		if p.CurrentTokenKind() != lexer.RightCurlyBrace {
			p.Expect(lexer.EndOfStatement, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return ast.MapLiteral{Entries: entries}
}

func (p *Parser) ParseList() ast.ListLiteral {
	items := []ast.Expression{}
	p.Expect(lexer.LeftBracket)
	for p.WhileNotEndOr(lexer.RightBracket) {
		items = append(items, p.ParseExpression(LogicalBindingPower))
		if p.IsNotCurrentlyEndOr(lexer.RightBracket) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightBracket)
	return ast.ListLiteral{Items: items}
}

func (p *Parser) ParseIndexExpression(left ast.Node, bp BindingPower) ast.IndexExpression {
	computed := p.Advance().Kind == lexer.LeftBracket
	var item ast.Expression
	if !computed {
		// Allow use of keywords as fields
		if !p.isMapIdentifier() {
			errors.ExpectedTokenError(lexer.Identifier, p.CurrentToken())
		}
		item = ast.Symbol{Identifier: p.Advance().Source}
	} else {
		item = p.ParseExpression(bp)
		p.Expect(lexer.RightBracket)
	}
	return ast.IndexExpression{
		Object:   left,
		Property: item,
		Computed: computed,
	}
}

func (p *Parser) ParseCallExpression(left ast.Node, bp BindingPower) ast.CallExpression {
	p.Expect(lexer.LeftParenthesis)
	args := []ast.CallParam{}
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		arg := ast.CallParam{}
		if p.CurrentTokenKind() == lexer.Colon {
			// Shorthand label if name and variable/field matches
			// 	person := Person()
			//	person2.greet(:person)
			// Equal to:
			// 	person2.greet(person: person)
			p.Advance()
			sym, isOk := p.ParseExpression(CallBindingPower), false
			switch sym := sym.(type) {
			case ast.Symbol:
				arg.Label = sym.Identifier
				arg.Value = sym
				isOk = true
			case ast.IndexExpression:
				if prop, ok := sym.Property.(ast.Symbol); ok {
					arg.Label = prop.Identifier
					arg.Value = sym
					isOk = true
				}
			}
			if !isOk {
				panic(errors.NewNodeError(errors.ErrInvalidLabelShorthand, sym))
			}
		} else {
			expr := p.ParseExpression(LogicalBindingPower)
			arg.Value = expr
			if expr, ok := expr.(ast.Symbol); ok && p.CurrentTokenKind() == lexer.Colon {
				// Label
				p.Advance()
				arg.Label = expr.Identifier
				arg.Value = p.ParseExpression(LogicalBindingPower)
			}
		}
		args = append(args, arg)
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	return ast.CallExpression{Callee: left, Args: args}
}

func (p *Parser) ParseEnumLiteral() ast.EnumLiteral {
	p.Expect(lexer.Dot)
	return ast.EnumLiteral{Name: p.Expect(lexer.Identifier).Source}
}

func (p *Parser) ParseLambdaExpression(left ast.Node, bp BindingPower) ast.LambdaExpression {
	p.Expect(lexer.Arrow)
	l := ast.LambdaExpression{}
	switch left := left.(type) {
	case ast.Symbol:
		l.Params = append(l.Params, ast.TypePair{Key: left.Identifier})
	case ast.ParamTuple:
		l.Params = left.Params
	case ast.TupleLiteral:
		for _, param := range left.Values {
			if _, ok := param.(ast.Symbol); !ok {
				panic(errors.NewNodeError(errors.ErrExpectedParamInLambda, param))
			}
			l.Params = append(l.Params, ast.TypePair{
				Key: param.(ast.Symbol).Identifier,
			})
		}
	default:
		panic(errors.NewNodeError(errors.ErrExpectedParamInLambda, left))
	}
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		l.Body = p.ParseBlock()
	} else {
		l.ExprBody = p.ParseExpression(CommaBindingPower)
	}
	return l
}