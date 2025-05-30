package parser

import (
	"slices"

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
	case lexer.Colon:
		// Type tuple
		return nil
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
		p.Error(errors.ExpectedToken(lexer.RightParenthesis, next))
		p.Advance()
		return ast.BadExpression{}
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
			errors.ExpectedToken(lexer.Identifier, p.CurrentToken())
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
				p.Error(errors.Node(errors.ErrInvalidLabelShorthand, sym))
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

func (p *Parser) ParseLambda(left ast.Node, bp BindingPower) (l ast.LambdaExpression) {
	p.Expect(lexer.Arrow)
	switch left := left.(type) {
	case ast.Symbol:
		l.Params = append(l.Params, ast.TypePair{Key: left.Identifier})
	case ast.ParamTuple:
		l.Params = left.Params
	case ast.TupleLiteral:
		for _, param := range left.Values {
			if _, ok := param.(ast.Symbol); !ok {
				p.Error(errors.Node(errors.ErrExpectedParamInLambda, param))
			}
			l.Params = append(l.Params, ast.TypePair{
				Key: param.(ast.Symbol).Identifier,
			})
		}
	default:
		p.Error(errors.Node(errors.ErrExpectedParamInLambda, left))
	}
	if p.CurrentTokenKind() == lexer.LeftCurlyBrace {
		l.Body = p.ParseBlock()
	} else {
		l.ExprBody = p.ParseExpression(DefaultBindingPower)
	}
	return l
}

func (p *Parser) ParseLeftRest() ast.RestExpression {
	p.Expect(lexer.Ellipsis)
	return ast.RestExpression{
		Left: true,
		Expr: p.ParseExpression(UnaryBindingPower),
	}
}

func (p *Parser) ParseRange(left ast.Node, bp BindingPower) ast.Expression {
	p.Advance() // ...
	l := left.(ast.Expression)
	// Rest if no expression on the right: [items...]
	if !slices.Contains(IsHandledNUD, p.CurrentTokenKind()) {
		return ast.RestExpression{Left: false, Expr: l}
	}
	// Range operator
	rang := ast.RangeExpression{
		Start: l,
		End:   p.ParseExpression(bp),
	}
	if p.CurrentTokenKind() == lexer.Ellipsis {
		// Step
		p.Advance()
		rang.Step = p.ParseExpression(bp)
	}
	return rang
}

func (p *Parser) ParsePipeline(left ast.Node, bp BindingPower) ast.PipelineExpression {
	steps := make([]ast.Node, 1, 2)
	steps[0] = left // First step

loop:
	for p.CurrentTokenKind() == lexer.Pipeline {
		p.Advance()
		switch p.CurrentTokenKind() {
		// Return should be the last step, without parameters, and should
		// only be used in expression statements
		case lexer.Return:
			steps = append(steps, ast.ReturnStatement{})
			break loop
		// Same thing for next
		case lexer.Next:
			steps = append(steps, ast.NextStatement{})
			break loop
		default:
			steps = append(steps, p.ParseExpression(bp))
		}
	}
	return ast.PipelineExpression{Steps: steps}
}
