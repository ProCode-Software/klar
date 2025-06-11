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

func (p *Parser) ParseParenExpression() ast.Expression {
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
		// Type tuple (for lambda)
		typeTuple := ast.TypeTuple{}
		if expr, ok := expr.(ast.Symbol); ok {
			p.Advance()
			typeTuple.Params = append(typeTuple.Params, ast.TypePair{
				expr.Identifier,
				p.ParseType(DefaultTypeBindingPower),
			})
		} else {
			// Expected identifier
		}
		parseSeries(
			p, &typeTuple.Params,
			func() ast.TypePair {
				key := p.Expect(lexer.Identifier).Source
				p.Expect(lexer.Colon)
				typ := p.ParseType(DefaultTypeBindingPower)
				return ast.TypePair{key, typ}
			},
			lexer.RightParenthesis, lexer.Comma, false,
		)
		return typeTuple
	case lexer.Comma:
		// Tuple (requires at least one comma)
		tuple := ast.TupleLiteral{}
		p.Advance()
		parseSeriesWithBP(
			p, &tuple.Values, ExpressionBindingPower,
			lexer.RightParenthesis, lexer.Comma,
		)
		return tuple
	case lexer.RightParenthesis:
		// Grouped expression
		p.Advance()
		return ast.ParenExpression{Expr: expr}
	default:
		p.Expect(lexer.RightParenthesis)
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

// Parses an index or slice expression.
//
//	list[1]
//	person.name
//	list[1:3]
func (p *Parser) ParseIndexExpression(left ast.Node, bp BindingPower) ast.Expression {
	computed := p.Advance().Kind == lexer.LeftBracket
	var item ast.Expression
	if !computed {
		// Allow use of keywords as fields
		item = ast.Symbol{Identifier: p.expectNonNumericMapIdent().Source}
	} else {
		var (
			left, right ast.Expression
			isSlice     bool
		)
		// Slice [:3]
		if p.CurrentTokenKind() == lexer.Colon {
			isSlice = true
			p.Advance()
		}
		item = p.ParseExpression(ExpressionBindingPower)
		if isSlice {
			right = item
		} else if p.CurrentTokenKind() == lexer.Colon {
			isSlice = true
			left = item
			p.Advance()
			// Slice [1:]
			if p.CurrentTokenKind() != lexer.RightBracket {
				right = p.ParseExpression(ExpressionBindingPower)
			}
		}
		p.Expect(lexer.RightBracket)
		if isSlice {
			return ast.SliceExpression{
				Object: left,
				Index:  left,
				Length: right,
			}
		}
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
			if p.Peek().Kind == lexer.Colon {
				// Label (allow keywords)
				if !slices.Contains(ast.ReservedIdent, p.CurrentTokenKind()) ||
					p.CurrentTokenKind() != lexer.Identifier {
					p.Error(errors.Token(errors.ErrInvalidLabel, p.CurrentToken()))
				}
				arg.Label = p.Advance().Source
				p.Advance() // :
			}
			arg.Value = p.ParseExpression(ExpressionBindingPower)
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
	case ast.TypeTuple:
		l.Params = left.Params
	case ast.TupleLiteral:
		for _, param := range left.Values {
			switch param := param.(type) {
			case ast.Symbol:
				l.Params = append(l.Params, ast.TypePair{Key: param.Identifier})
			// Allow (_, b) -> ...
			case ast.Discard:
				l.Params = append(l.Params, ast.TypePair{Key: "_"})
			default:
				p.Error(errors.Node(errors.ErrExpectedParamInLambda, param))
			}
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
	// Allow [...] in when case
	// But not ..._
	if p.CurrentTokenKind() == lexer.Underscore {
		p.Error(errors.Token(errors.ErrUnderscoreWithRest, p.CurrentToken()))
	}
	if p.isWhenCase && !slices.Contains(IsHandledNUD, p.CurrentTokenKind()) {
		return ast.RestExpression{Left: true}
	}
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
		if _, ok := left.(ast.Discard); ok {
			// _... not allowed
			p.Error(errors.Node(errors.ErrUnderscoreWithRest, left))
		}
		return ast.RestExpression{Left: false, Expr: l}
	}
	// Range operator
	rang := ast.RangeExpression{
		From: l,
		To:   p.ParseExpression(bp),
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

func (p *Parser) parseCaseSubExpr() ast.Expression {
	tok := p.CurrentTokenKind()
	switch tok {
	// Relational operators don't need explicit LHS
	// 	when x {
	// 		< 5 -> ...
	// }
	case lexer.EqualEqual, lexer.NotEqual, lexer.LessThan, lexer.GreaterThan,
		lexer.GreaterEqualTo, lexer.LessEqualTo, lexer.In:
		return p.ParseBinaryExpression(nil, BindingPowerMap[tok])
	case lexer.Question:
		p.Advance()
		return ast.NilLiteral{Shorthand: true}
	case lexer.Underscore:
		p.Advance()
		return ast.Discard{}
	default:
		return p.ParseExpression(LambdaBindingPower) // Don't include -> (lambda)
	}
}

func (p *Parser) parseWhenCase(subjects int) ast.WhenCase {
	var (
		c        = ast.WhenCase{}
		commaExp = make([]ast.Expression, 0, subjects)
		orOpts   [][]ast.Expression
	)
	p.isWhenCase = true
	// ',' binds tighter than '|' in case
loop:
	for p.HasTokens() {
		if p.IsCurrently(lexer.When, lexer.Arrow, lexer.EndOfStatement) {
			break loop
		}
		commaExp = append(commaExp, p.parseCaseSubExpr())
		switch p.CurrentTokenKind() {
		case lexer.Stroke:
			orOpts = append(orOpts, commaExp)
			clear(commaExp)
			commaExp = commaExp[:0]
			p.Advance()
		case lexer.When, lexer.Arrow:
			orOpts = append(orOpts, commaExp)
			break loop
		case lexer.Comma:
			p.Advance()
		default:
			p.Expect(lexer.Arrow)
		}
	}
	c.Options = orOpts
	// Guard clause
	// 	when x, y {
	//		5, _ when y < 10 -> ...
	// 	}
	if p.CurrentTokenKind() == lexer.When {
		p.Advance()
		p.isWhenGuard = true
		c.Guard = p.ParseExpression(LambdaBindingPower)
		p.isWhenGuard = false
	}
	p.isWhenCase = false
	p.Expect(lexer.Arrow)
	switch p.CurrentTokenKind() {
	case lexer.LeftCurlyBrace:
		c.Body = p.ParseBlock()
		c.InBraces = true
	default:
		res := p.ParseLED(DefaultBindingPower)
		switch res := res.(type) {
		// Allow some kinds of statements outside of braces
		case ast.AssignmentStatement, ast.ReturnStatement,
			ast.NextStatement, ast.UpdateStatement:
			c.Body = []ast.Statement{res.(ast.Statement)}
		// All expressions are allowed
		case ast.Expression:
			c.BodyExpr = res
		default:
			// Expected expression error
			p.errExpectedExpr(res)
			c.BodyExpr = ast.BadExpression{}
		}
	}
	return c
}

func (p *Parser) ParseWhenBlock() ast.WhenExpression {
	p.Expect(lexer.When)
	w := ast.WhenExpression{}
	if p.CurrentTokenKind() != lexer.LeftCurlyBrace {
		// Subjects
		parseSeriesWithBP(
			p, &w.Subjects, ExpressionBindingPower,
			lexer.LeftCurlyBrace, lexer.Comma,
		)
	} else {
		p.Expect(lexer.LeftCurlyBrace)
	}
	lenSubj := len(w.Subjects)
	parseSeries(
		p, &w.Cases,
		func() ast.WhenCase { return p.parseWhenCase(lenSubj) },
		lexer.RightCurlyBrace, lexer.EndOfStatement, true,
	)
	return w
}
