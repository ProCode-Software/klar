package parser

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseBinaryExpression(left ast.Expression, bp BindingPower) *ast.BinaryExpression {
	op := p.Advance()
	if p.CurrKind() == lexer.Newline {
		p.Advance()
	}
	right := p.ParseExpression(bp)
	return &ast.BinaryExpression{
		Left:     left,
		Operator: newOperator(op),
		Right:    right,
	}
}

// Currently, '-' and '!' are the only unary operators.
func (p *Parser) ParseUnaryExpression() *ast.UnaryExpression {
	op := p.Advance()
	right := p.ParseExpression(UnaryBindingPower)
	return &ast.UnaryExpression{Operator: newOperator(op), Right: right}
}

const (
	dirNeutral = iota
	dirLessThan
	dirGreaterThan
)

func (p *Parser) ParseRelationalExpression(
	left ast.Expression, bp BindingPower,
) *ast.RelationalExpression {
	rel := &ast.RelationalExpression{}
	rel.Expressions = append(rel.Expressions, left) // First expression
	dir := dirNeutral
loop:
	for {
		switch p.CurrKind() {
		case lexer.NotEqual:
			if len(rel.Operators) >= 1 { // Allow 'a != b' but not 'a != b != c'
				err := klarerrs.Token(klarerrs.ErrChainedNotEqual, p.Curr())
				err.Hint(
					"In 'a != b != c', 'a' could still be equal to 'c'. Since this " +
						"is confusing, chaining the '!=' operator isn't allowed in Klar.\n\n" +
						"* To check if all values are different from each other, use " +
						"'a != b && b != c && a != c'.\n" + "* Otherwise, split the chain into " +
						"multiple comparisons: 'a != b && b != c' if this is intentional.",
				)
				p.ErrorLabelled(err, "Can't chain '!='")
			}
			fallthrough
		case lexer.EqualEqual:
			// Hint for use of JavaScript === or !==
			if p.PeekKind() == lexer.Equal {
				p.ErrorLabelled(klarerrs.Range(klarerrs.ErrTripleEqual, ranges.Range{
					Start: p.Curr().Position,
					End:   p.Advance().Position.Add(0, 3),
				}).SetParam("op", p.CurrKind()), "Remove the last '=' character")
			}
		// Check for multidirectional comparisons (</<= with >/>=)
		case lexer.GreaterThan, lexer.GreaterEqualTo:
			if dir == dirLessThan {
				p.multidirCompareErr(rel.Operators, p.CurrKind())
			}
			dir = dirGreaterThan
		case lexer.LessThan, lexer.LessEqualTo:
			if dir == dirGreaterThan {
				p.multidirCompareErr(rel.Operators, p.CurrKind())
			}
			dir = dirLessThan
		default:
			break loop // Non-relational operator
		}
		rel.Operators = append(rel.Operators, newOperator(p.Advance()))
		rel.Expressions = append(rel.Expressions, p.ParseExpression(bp))
	}
	return rel
}

func (p *Parser) multidirCompareErr(ops []ast.Operator, got lexer.TokenType) {
	err := klarerrs.Token(klarerrs.ErrMultiDirectionCompareChain, p.Curr())
	var next lexer.TokenType
	switch got {
	case lexer.GreaterThan:
		next = lexer.LessThan
	case lexer.GreaterEqualTo:
		next = lexer.LessEqualTo
	case lexer.LessThan:
		next = lexer.GreaterThan
	case lexer.LessEqualTo:
		next = lexer.GreaterEqualTo
	}
	if len(ops) == 1 { // 3 operands
		err.Hintf(
			"Reorder the comparison: (e.g. 'a %s c %s b')\n"+
				"Or, split it into multiple comparisons: (e.g. 'a %[1]s b && b %[3]s c')",
			ops[0], next, got,
		)
	} else {
		err.Hint(
			"Reorder the comparison, or split it into multiple comparisons" +
				" (e.g. 'a < b > c' to 'a < b && b > c')",
		)
	}
	p.ErrorLabelled(err, klarerrs.Quote(next.String())+" must be used")
}

func (p *Parser) ParseParenExpression() ast.Expression {
	p.Advance() // (
	if p.CurrKind() == lexer.RightParenthesis {
		// Empty tuple
		p.Advance()
		return &ast.TupleLiteral{}
	}
	expr := p.ParseExpression(ExpressionBindingPower)
	if p.CurrKind() != lexer.Comma {
		// Grouped expression
		p.Expect(lexer.RightParenthesis)
		return &ast.ParenExpression{Expression: expr}
	}

	// Tuple (requires at least one comma)
	p.Advance() // ,
	tuple := &ast.TupleLiteral{Values: []ast.Expression{expr}}
	for p.WhileNot(lexer.RightParenthesis) {
		tuple.Values = append(tuple.Values, p.ParseExpression(ExpressionBindingPower))
		if p.CurrKind() != lexer.RightParenthesis {
			p.Expect(
				lexer.Comma,
				noAdvance, withMessage("between tuple items"),
				withLabel("Expected a comma after this item"),
			)
			if p.CurrKind() == lexer.Newline {
				p.Advance() // Missing comma
			}
		}
	}
	p.Expect(lexer.RightParenthesis, noAdvance)
	// TODO: better message for missing ','
	return tuple
}

func (p *Parser) ParseMap() *ast.MapLiteral {
	p.Expect(lexer.HashLeftCurlyBrace)
	var entries []*ast.MapItem
	for p.WhileNotEndOr(lexer.RightCurlyBrace) {
		// Shorthand: #{ :name } = #{ name: name }
		if p.CurrKind() == lexer.Colon {
			start := p.Advance()
			key, val := p.expectShorthand()
			entries = append(entries, &ast.MapItem{
				Keys:      []ast.Expression{key},
				ColonPos:  start.Position,
				Value:     val,
				Shorthand: true,
				BaseNode:  newBaseNode(start.Position, val.GetRange().End),
			})
		} else {
			// Normal properties: quotes not required for non-reserved string key
			entry := &ast.MapItem{}
			entry.Range.Start = p.Curr().Position

			// Keys and possibly a rest
			for p.HasTokens() {
				item := p.ParseExpression(ExpressionBindingPower)
				if rest, ok := item.(*ast.RestExpression); ok {
					if len(entry.Keys) > 0 {
						p.ErrorLabelled(
							klarerrs.Slice(klarerrs.ErrMultipleKeysInMapRest, entry.Keys),
							"Only 1 key is allowed in a rest",
						)
					}
					entry.Keys = nil
					entry.Value = rest
					entry.Rest = true
					break
				}
				entry.Keys = append(entry.Keys, item)
				if p.CurrKind() != lexer.Comma {
					break
				}
				p.Advance()
			}
			// Value
			if !entry.Rest {
				entry.ColonPos = p.Expect(lexer.Colon).Position
				entry.Value = p.ParseExpression(ExpressionBindingPower)
			}
			markEndPos(p, entry)
			entries = append(entries, entry)
		}
		curr := p.CurrKind()
		// Known issue: required comma after ... because ParseExpression parses
		// anything after it as a range expression. It can't be prevented here.
		// TODO: maybe fix?
		if curr == lexer.Colon && p.Curr().Line > p.PeekBehind().Line {
			continue
		}
		if curr != lexer.RightCurlyBrace {
			p.ExpectOneOf(lexer.Newline, lexer.Comma)
		}
	}
	p.Expect(lexer.RightCurlyBrace)
	return &ast.MapLiteral{Entries: entries}
}

func (p *Parser) ParseList() *ast.ListLiteral {
	var items []ast.Expression
	p.Expect(lexer.LeftBracket)
	parseExprSeries(p, &items, ExpressionBindingPower, lexer.RightBracket, lexer.Comma)
	return &ast.ListLiteral{Items: items}
}

// Parses an index or slice expression.
//
//	list[0]    list.first
//	list[1...3]  list[1:]
//	list[..<3]   list[:]
func (p *Parser) ParseIndexExpression(left ast.Expression, bp BindingPower) ast.Expression {
	var item ast.Expression
	if p.Advance().Kind != lexer.LeftBracket {
		// Allow use of keywords as fields
		return &ast.IndexExpression{
			Object:   left,
			Property: p.ParseMapIdentifier(0).Symbol(),
			Computed: false,
		}
	}
	// Slice with no explicit start bound [..<3] or [...3]
	if k := p.CurrKind(); k == lexer.Ellipsis || k == lexer.DotDotLessThan {
		s := &ast.SliceExpression{Object: left, Operator: newOperator(p.Advance())}
		if p.CurrKind() != lexer.RightBracket {
			s.To = p.ParseExpression(RangeBindingPower)
		} else if k == lexer.DotDotLessThan {
			// '..<' must have end
			p.ErrorLabelled(
				klarerrs.Token(klarerrs.ErrExpectedExprAfterOpenRange, p.PeekBehind()),
				"Expected an upper bound after this",
			)
		}
		if p.CurrKind() == lexer.Ellipsis {
			p.ErrorLabelled(
				klarerrs.Token(klarerrs.ErrStepInListSlice, p.Advance()),
				"List slices must be continuous",
			)
			_ = p.ParseExpression(RangeBindingPower)
		}
		p.Expect(lexer.RightBracket)
		return s
	}
	// Expression
	item = p.ParseExpression(ExpressionBindingPower)
	p.Expect(lexer.RightBracket)

	switch rang := item.(type) {
	case *ast.RangeExpression:
		if rang.Step != nil {
			p.ErrorLabelled(
				klarerrs.Node(klarerrs.ErrStepInListSlice, rang.Step),
				"List slices must be continuous",
			)
		}
		return &ast.SliceExpression{
			Object:   left,
			From:     rang.From,
			To:       rang.To,
			Operator: rang.Operator,
		}
	case *ast.RestExpression:
		return &ast.SliceExpression{
			Object:   left,
			From:     rang.Expression,
			Operator: ast.Operator{lexer.Ellipsis, rang.Range.End.Sub(0, 3)},
		}
	}
	return &ast.IndexExpression{
		Object:   left,
		Property: item,
		Computed: true,
	}
}

func (p *Parser) ParseCallExpression(left ast.Expression, bp BindingPower) *ast.CallExpression {
	p.Expect(lexer.LeftParenthesis)
	switch left := left.(type) {
	case *ast.ParenExpression:
		if left, ok := left.Expression.(*ast.LambdaExpression); ok {
			p.Error(klarerrs.Node(klarerrs.ErrSelfExecFunc, left))
		}
	case *ast.LambdaExpression:
		p.Error(klarerrs.Node(klarerrs.ErrSelfExecFunc, left))
	}
	var args []*ast.CallParam
	for p.WhileNotEndOr(lexer.RightParenthesis) {
		arg := &ast.CallParam{}
		arg.Range.Start = p.Curr().Position
		switch {
		case p.CurrKind() == lexer.Colon:
			// Shorthand label if name and variable/field matches
			// 	person := Person()
			//	person2.greet(:person)
			// Equal to:
			// 	person2.greet(person: person)
			p.Advance()
			key, val := p.expectShorthand()
			arg.Label, arg.Value = new(key.ToIdentifier()), val
		case p.PeekKind() == lexer.Colon:
			// Label (allow keywords)
			arg.Label = new(p.ParseMapIdentifier(isLabel))
			p.Advance() // :
			fallthrough
		default:
			arg.Value = p.ParseExpression(ExpressionBindingPower)
		}
		markEndPos(p, arg)
		args = append(args, arg)
		if p.IsNotCurrentlyEndOr(lexer.RightParenthesis) {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)
	return &ast.CallExpression{Callee: left, Args: args}
}

func (p *Parser) ParseEnumLiteral() ast.Expression {
	p.Expect(lexer.Dot)
	if p.CurrKind() == lexer.LeftParenthesis {
		return p.ParseStructDotInit()
	}
	return &ast.EnumLiteral{Name: p.ParseMapIdentifier(0)}
}

func (p *Parser) ParseStructDotInit() *ast.StructDotInit {
	// Parsing starts with (
	call := p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis))
	return &ast.StructDotInit{Params: call.Args}
}

func (p *Parser) ParseLambda() *ast.LambdaExpression {
	l := &ast.LambdaExpression{}
	p.Advance() // func
	switch p.CurrKind() {
	case lexer.LeftParenthesis:
		// Params and optional type/default in parens
		p.Advance()
		if p.CurrKind() != lexer.RightParenthesis {
			p.parseAssignableTypePairs(&l.Params, nil, false)
		}
		l.InParen = true
		p.Expect(lexer.RightParenthesis)
	case lexer.Arrow, lexer.LeftCurlyBrace:
	default:
		parseSeries(p, &l.Params, func() *ast.AssignableTypePair {
			d := &ast.AssignableTypePair{Keys: []ast.Assignable{p.ParseAssignable()}}
			// Non-parenthesized type
			if p.CurrKind() == lexer.Colon {
				p.ErrorLabelled(
					klarerrs.Token(klarerrs.ErrParenAroundLambdaType, p.Advance()),
					"This parameter must be in parentheses",
				)
				d.Type = p.ParseType(DefaultTypeBindingPower) // Still parse it
			}
			// Non-parenthesized default
			if c := p.CurrKind(); c == lexer.Equal || c == lexer.ColonEqual {
				p.ErrorLabelled(
					klarerrs.Token(klarerrs.ErrParenAroundLambdaDefault, p.Advance()),
					"This parameter must be in parentheses",
				)
				d.Value = p.ParseExpression(ExpressionBindingPower) // Still parse it
			}
			return d
		}, 0, lexer.Comma, false)
	}
	switch p.CurrKind() {
	case lexer.Arrow:
		p.Advance()
		l.Expr = p.ParseExpression(ExpressionBindingPower)
	case lexer.LeftCurlyBrace:
		l.Block = p.ParseBlock()
	default:
		p.ErrorLabelled(
			klarerrs.ExpectedToken(lexer.LeftCurlyBrace, p.Curr()),
			"Expected a block or an arrow '->'",
		)
	}
	return l
}

// When case only: [...] or `..."string"`
func (p *Parser) ParseLeftRest() *ast.RestExpression {
	p.Expect(lexer.Ellipsis)
	var expr ast.Expression
	if nud, ok := p.handleNUD(p.CurrKind()); ok {
		expr = p.ParseLED(nud, UnaryBindingPower)
	}
	return &ast.RestExpression{Left: true, Expression: expr}
}

func (p *Parser) ParseRange(left ast.Expression, bp BindingPower) ast.Expression {
	op := p.Advance() // ... or ..<
	if right, handled := p.handleNUD(p.CurrKind()); handled {
		// Range operator
		rang := &ast.RangeExpression{
			From:     left,
			To:       p.ParseLED(right, bp),
			Operator: newOperator(op),
		}
		curr := p.CurrKind()
		if curr == lexer.DotDotLessThan {
			p.ErrorLabelled(
				klarerrs.Token(klarerrs.ErrEllipsisForOpenRangeStep, p.Curr()),
				"Steps are defined using '...'",
			)
			curr = lexer.Ellipsis
		}
		if curr == lexer.Ellipsis {
			// Step
			p.Advance()
			rang.Step = p.ParseExpression(bp)
		}
		return rang
	}
	if op.Kind == lexer.DotDotLessThan {
		// Expression required
		p.ErrorLabelled(
			klarerrs.Token(klarerrs.ErrExpectedExprAfterOpenRange, op),
			"Open ranges must have an upper bound",
		)
	}
	// Rest if no expression on the right: [items...]
	if _, ok := left.(*ast.Discard); ok {
		// _... not allowed
		p.ErrorLabelled(
			klarerrs.Node(klarerrs.ErrUnderscoreWithRest, left),
			"Remove this discard",
		)
	}
	return &ast.RestExpression{Expression: left}
}

func (p *Parser) ParsePipeline(left ast.Expression, bp BindingPower) *ast.PipelineExpression {
	returnIndex := -1
	steps := []ast.Node{left} // First step

	for p.CurrKind() == lexer.Pipeline {
		p.Advance()
		// Return in a pipeline returns the previous result.
		// Return should be the last step, without parameters, and should
		// only be used in expression statements
		if p.CurrKind() == lexer.Return {
			returnIndex = len(steps)
			steps = append(steps, p.ParseStatement(noEOS))
			continue
		}
		steps = append(steps, p.ParseExpression(bp))
	}
	// Return must be the last step. The type checker will also make sure this
	// pipeline is not used as an expression.
	if returnIndex >= 0 && returnIndex != len(steps)-1 {
		p.ErrorLabelled(
			klarerrs.Node(klarerrs.ErrReturnPipelineNotLast, steps[returnIndex]),
			"'return' must be the last step",
		)
	}
	return &ast.PipelineExpression{Steps: steps}
}

// The version is validated when the attribute is evaluated (analysis-time).
func (p *Parser) ParseVersion() ast.Expression {
	var b strings.Builder
	skipNewline := func() {
		if p.CurrKind() == lexer.Newline {
			p.Advance()
		}
	}

	// First part should already be validated
	b.WriteString(p.Advance().Source)
	for p.CurrKind() == lexer.Dot {
		p.Advance()
		b.WriteByte('.')
		b.WriteString(p.Expect(lexer.Numeric).Source)
		skipNewline()
	}
	skipNewline()
	// Tag: v1.0 beta
	if p.CurrKind() == lexer.Identifier {
		b.WriteByte(' ')
		b.WriteString(p.Advance().Source)
		skipNewline()
		// Number after the tag: v1.0 beta 2
		if p.CurrKind() == lexer.Numeric {
			b.WriteByte(' ')
			b.WriteString(p.Advance().Source)
		}
	}
	ver := &ast.VersionLiteral{Version: b.String()}
	return ver
}

func (p *Parser) ParseListCast() *ast.ListCastExpression {
	p.Expect(lexer.LeftBracket)
	typ := p.ParseType(DefaultTypeBindingPower)
	p.Expect(lexer.RightBracket)
	return &ast.ListCastExpression{
		Type: typ,
		Args: p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis)).Args,
	}
}

func (p *Parser) ParseMapCast() *ast.MapCastExpression {
	p.Expect(lexer.HashLeftCurlyBrace)
	key := p.ParseType(DefaultTypeBindingPower)
	p.Expect(lexer.Colon)
	val := p.ParseType(DefaultTypeBindingPower)
	p.Expect(lexer.RightCurlyBrace)
	return &ast.MapCastExpression{
		KeyType: key, ValueType: val,
		Args: p.ParseCallExpression(nil, bpOf(lexer.LeftParenthesis)).Args,
	}
}

func (p *Parser) ParseObjectPipeline(obj ast.Expression, bp BindingPower) *ast.ObjectPipeline {
	pipeline := &ast.ObjectPipeline{Object: obj}
	for p.CurrKind() == lexer.StrokeDot {
		p.Advance() // |.
		var lhs ast.Expression
		// Computed index: |. [0]
		if p.CurrKind() == lexer.LeftBracket {
			start := p.Advance().Position
			lhs = p.ParseIndexExpression(nil, bpOf(lexer.LeftParenthesis))
			markStartEndPos(p, lhs, start)
		} else {
			// Must be symbol
			if isValidIdentifier(p.Curr().Kind) {
				lhs = p.ParseValidIdent().Symbol()
			} else {
				p.Error(klarerrs.Node(klarerrs.ErrInvalidObjectPipeStep, lhs))
				lhs = &ast.BadExpression{Value: lhs}
			}
		}
		// Index or call
		if k := p.CurrKind(); !isAssignment(k) && k != lexer.StrokeDot {
			lhs = p.ParseLED(lhs, bp)
		}
		// Assignment
		if k := p.CurrKind(); isAssignment(k) && k != lexer.ColonEqual {
			l := p.validateAssignable(lhs)
			assg := &ast.AssignmentStatement{
				Assignee: []ast.Assignable{l},
				Operator: newOperator(p.Advance()),
				Values:   []ast.Expression{p.ParseExpression(bp)},
			}
			markStartEndPos(p, assg, l.GetRange().Start)
			pipeline.Steps = append(pipeline.Steps, assg)
		} else {
			// Validate method call
			if _, ok := lhs.(*ast.CallExpression); !ok {
				p.Error(klarerrs.Node(klarerrs.ErrInvalidObjectPipeStep, lhs))
				lhs = &ast.BadExpression{Value: lhs}
			}
			pipeline.Steps = append(pipeline.Steps, lhs)
		}
	}
	return pipeline
}

func (p *Parser) ParseForExpression() *ast.ForExpression {
	p.Advance() // for
	f := &ast.ForExpression{}
	f.Variables, f.Iterator = p.parseForVariables()
	k := p.CurrKind()
	switch {
	case p.isEqual(p.Curr()):
		// = or :=; neither are allowed
		fallthrough
	default:
		p.Error(klarerrs.Token(klarerrs.ErrInvalidForExprOperator, p.Curr()))
		p.AdvanceNonBoundary()
		// TODO: should we still parse an expression after?
	case isAssignment(k), k == lexer.Arrow:
		// -> or any assignment except := or =
		f.Operator = newOperator(p.Advance())
		// Allow spread (...) to be included at the end, to spread entire loop.
		f.Value = p.ParseExpressionFilter(excludeIf(lexer.Ellipsis), RangeBindingPower, try)
	case k == lexer.LeftCurlyBrace:
		f.Block = p.ParseBlock()
	}
	return f
}

func (p *Parser) ParseGoExpression() *ast.GoExpression {
	p.Advance() // go
	if p.CurrKind() == lexer.LeftCurlyBrace {
		return &ast.GoExpression{Body: p.ParseBlock()}
	} else {
		g := &ast.GoExpression{Expression: p.ParseExpression(UnaryBindingPower)}
		if _, ok := g.Expression.(*ast.CallExpression); !ok {
			p.ErrorLabelled(
				klarerrs.Node(klarerrs.ErrMustBeFuncCall, g.Expression).
					SetParam("expr", lexer.Go),
				"This must be a function call",
			)
		}
		return g
	}
}

func (p *Parser) ParseAwaitExpression() *ast.AwaitExpression {
	p.Advance() // await
	return &ast.AwaitExpression{Expression: p.ParseExpression(UnaryBindingPower)}
}

func (p *Parser) ParseTryExpression() *ast.TryExpression {
	p.Advance() // try
	t := &ast.TryExpression{}
	// Invalid try-catch block: try {}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		p.ErrorLabelled(
			klarerrs.Token(klarerrs.ErrTryBlock, p.Curr()),
			"Klar doesn't have try-catch",
		)
		p.ParseBlock() // Just parse it
		return t
	}
	t.Expression = p.ParseExpression(UnaryBindingPower)
	if _, ok := t.Expression.(*ast.CallExpression); !ok {
		p.ErrorLabelled(
			klarerrs.Node(klarerrs.ErrMustBeFuncCall, t.Expression).
				SetParam("expr", lexer.Try),
			"This must be a function call",
		)
	}
	return t
}

func (p *Parser) ParseAssertExpression(left ast.Expression) *ast.AssertExpression {
	p.Advance() // !!
	return &ast.AssertExpression{Expression: left}
}
