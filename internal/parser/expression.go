package parser

import (
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/errors"
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

// Currently, '-' is the only unary operator.
func (p *Parser) ParseUnaryExpression() *ast.UnaryExpression {
	op := p.Advance()
	right := p.ParseExpression(UnaryBindingPower)
	return &ast.UnaryExpression{Operator: newOperator(op), Right: right}
}

func (p *Parser) ParseRelationalExpression(
	left ast.Expression, bp BindingPower,
) *ast.RelationalExpression {
	rel := &ast.RelationalExpression{}
	rel.Expressions = append(rel.Expressions, left) // First expression
	var gtLtDir uint8                               // 0: none; 1: < or <=; 2: > or >=
loop:
	for {
		switch p.CurrKind() {
		case lexer.NotEqual:
			if len(rel.Operators) >= 1 { // Allow 'a != b' but not 'a != b != c'
				err := errors.Token(errors.ErrChainedNotEqual, p.Curr())
				err.Hint(
					"In 'a != b != c', 'a' could still be equal to 'c'. Since this " +
						"is confusing, chaining the '!=' operator isn't allowed in Klar. " +
						"To check if all values are different from each other, use " +
						"'a != b && b != c && a != c'. Otherwise, split the chain into " +
						"multiple comparisons with '&&': 'a != b && b != c'",
				)
				p.Error(err)
			}
			fallthrough
		case lexer.EqualEqual:
			if gtLtDir != 0 {
				p.Error(errors.Token(errors.ErrInequalityWithEqualChain, p.Curr()))
			}
			// Hint for use of JavaScript === or !==
			if p.PeekKind() == lexer.Equal {
				p.Error(errors.Range(errors.ErrTripleEqual, ranges.FromPosition(
					p.Curr().Position,
					ranges.Add(p.Advance().Position, 0, 3),
				)).SetParam("op", p.CurrKind()))
			}
		// Check for multidirectional comparisons (</<= with >/>=)
		case lexer.GreaterThan, lexer.GreaterEqualTo:
			if gtLtDir == 1 {
				p.multidirCompareErr(rel.Operators, p.CurrKind())
			}
			gtLtDir = 2
		case lexer.LessThan, lexer.LessEqualTo:
			if gtLtDir == 2 {
				p.multidirCompareErr(rel.Operators, p.CurrKind())
			}
			gtLtDir = 1
		default:
			break loop // Non-relational operator
		}
		rel.Operators = append(rel.Operators, newOperator(p.Advance()))
		rel.Expressions = append(rel.Expressions, p.ParseExpression(bp))
	}
	return rel
}

func (p *Parser) multidirCompareErr(ops []ast.Operator, curr lexer.TokenType) {
	err := errors.Token(errors.ErrMultiDirectionCompareChain, p.Curr())
	if len(ops) == 1 { // 3 operands
		var next lexer.TokenType
		switch curr {
		case lexer.GreaterThan:
			next = lexer.LessThan
		case lexer.GreaterEqualTo:
			next = lexer.LessEqualTo
		case lexer.LessThan:
			next = lexer.GreaterThan
		case lexer.LessEqualTo:
			next = lexer.GreaterEqualTo
		}
		err.Hintf("Reorder the comparison: a %s c %s b\n"+
			"Or, split it into multiple chains with '&&': a %[1]s b && b %[3]s c",
			ops[0], next, curr,
		)
	} else {
		err.Hint("Reorder the comparison, or split it into multiple chains with '&&'" +
			" (e.g. 'a < b > c' to 'a < b && b > c')",
		)
	}
	p.Error(err)
}

func (p *Parser) ParseParamList() *ast.AssignableTuple {
	p.Expect(lexer.LeftParenthesis)
	tuple := &ast.AssignableTuple{}
	if p.CurrKind() != lexer.RightParenthesis {
		parseSeries(p, &tuple.Values, func() *ast.AssignableTypePair {
			pair := &ast.AssignableTypePair{}
			parseSeries(p, &pair.Keys, func() ast.Assignable { return p.ParseAssignable() }, 0, lexer.Comma, false)
			if p.CurrKind() == lexer.Colon {
				p.Advance()
				pair.Type = p.ParseType(DefaultTypeBindingPower)
			}
			if p.isEqual() {
				p.Advance()
				pair.Value = p.ParseExpression(ExpressionBindingPower)
			}
			return pair
		}, 0, lexer.Comma, false)
	}
	p.Expect(lexer.RightParenthesis)
	return tuple
}

func (p *Parser) ParseParenExpression() ast.Expression {
	p.Advance() // (
	if p.CurrKind() == lexer.RightParenthesis {
		// Empty tuple
		p.Advance()
		return &ast.TupleLiteral{}
	}
	expr := p.ParseExpression(ExpressionBindingPower)
	next := p.Curr()
	switch next.Kind {
	case lexer.Comma:
		// Tuple (requires at least one comma)
		tuple := &ast.TupleLiteral{Values: []ast.Expression{expr}}
		parseExprSeries(
			p, &tuple.Values, ExpressionBindingPower,
			lexer.RightParenthesis, lexer.Comma,
		)
		return tuple
	default:
		// Grouped expression
		p.Expect(lexer.RightParenthesis)
		return &ast.ParenExpression{Expression: expr}
	}
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
				Value:     val,
				Shorthand: true,
				BaseNode:  newBaseNode(start.Position, val.GetRange().End),
			})
		} else {
			// Normal properties: quotes not required for non-reserved string key
			entry := &ast.MapItem{}
			entry.Range.Start = p.Curr().Position
			parseExprSeries(p, &entry.Keys, ExpressionBindingPower, 0, lexer.Comma)

			// Spread #{ key: 1, values... }
			if rest, ok := entry.Keys[len(entry.Keys)-1].(*ast.RestExpression); ok {
				if len(entry.Keys) > 1 {
					// There must be exactly 1 key
					p.Error(errors.Slice(errors.ErrMultipleKeysInMapRest, entry.Keys))
					entry.Keys = entry.Keys[:len(entry.Keys)-1]
				}
				entry.Rest = true
				entry.Value = rest
			} else {
				p.Expect(lexer.Colon)
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
			p.Expect(lexer.Newline, lexer.Comma)
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
//	list[1:3]  list[1:]
//	list[:3]   list[:]
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
	// Slice [:3]
	if k := p.CurrKind(); k == lexer.Ellipsis || k == lexer.DotDotLessThan {
		r := &ast.RangeExpression{Operator: newOperator(p.Advance())}
		if p.CurrKind() != lexer.RightBracket {
			r.To = p.ParseExpression(RangeBindingPower)
		} else if k == lexer.DotDotLessThan {
			// '..<' must have end
			p.Error(errors.Token(errors.ErrExpectedExprAfterOpenRange, p.PeekBehind()))
		}
		if p.CurrKind() == lexer.Ellipsis {
			p.Error(errors.Token(errors.ErrStepInListSlice, p.Advance()))
			r.Step = p.ParseExpression(RangeBindingPower)
		}
		p.Expect(lexer.RightBracket)
		return r
	}
	// Expression
	item = p.ParseExpression(ExpressionBindingPower)
	p.Expect(lexer.RightBracket)

	if rang, ok := item.(*ast.RangeExpression); ok {
		if rang.Step != nil {
			p.Error(errors.Node(errors.ErrStepInListSlice, rang.Step))
		}
		return &ast.SliceExpression{
			Object:   left,
			From:     rang.From,
			To:       rang.To,
			Operator: rang.Operator,
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
			p.Error(errors.Node(errors.ErrSelfExecFuncNotAllowed, left))
		}
	case *ast.LambdaExpression:
		p.Error(errors.Node(errors.ErrSelfExecFuncNotAllowed, left))
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
			label := symbolToIdentifier(key)
			arg.Label, arg.Value = &label, val
		case p.PeekKind() == lexer.Colon:
			// Label (allow keywords)
			label := p.ParseMapIdentifier(isLabel)
			arg.Label = &label
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
	return &ast.EnumLiteral{Name: p.ParseIdentifier()}
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
			p.parseAssignableTypePairs(&l.Params)
		}
		l.InParen = true
		p.Expect(lexer.RightParenthesis)
	case lexer.Arrow, lexer.LeftCurlyBrace:
	default:
		parseSeries(p, &l.Params, func() *ast.AssignableTypePair {
			d := &ast.AssignableTypePair{Keys: []ast.Assignable{p.ParseAssignable()}}
			if p.CurrKind() == lexer.Colon {
				// Non-parenthesized type
				p.Error(errors.Token(errors.ErrParenAroundLambdaType, p.Advance()))
				d.Type = p.ParseType(DefaultTypeBindingPower) // Still parse it
			}
			if c := p.CurrKind(); c == lexer.Equal || c == lexer.ColonEqual {
				// Non-parenthesized default
				p.Error(errors.Token(errors.ErrParenAroundLambdaDefault, p.Advance()))
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
		p.Error(errors.ExpectedToken(lexer.LeftCurlyBrace, p.Curr()))
	}
	return l
}

// parseAssignableTypePairs parses a series of assignable expressions, optionally
// followed by an optional type and/or a default value. An item in pairs may
// have multiple keys and no type or value.
func (p *Parser) parseAssignableTypePairs(pairs *[]*ast.AssignableTypePair) {
	parseSeries(p, pairs, func() *ast.AssignableTypePair {
		pair := &ast.AssignableTypePair{}
		parseSeries(p, &pair.Keys, p.ParseAssignable, 0, lexer.Comma, false)
		if p.CurrKind() == lexer.Colon {
			p.Advance()
			pair.Type = p.ParseType(DefaultTypeBindingPower)
		}
		if p.isEqual() {
			p.Advance()
			pair.Value = p.ParseExpression(ExpressionBindingPower)
		}
		return pair
	}, 0, lexer.Comma, false)
}

// When case only: [...]
func (p *Parser) ParseLeftRest() *ast.RestExpression {
	p.Expect(lexer.Ellipsis)
	return &ast.RestExpression{Expression: p.ParseExpression(UnaryBindingPower)}
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
			p.Error(errors.Token(errors.ErrEllipsisForOpenRangeStep, p.Curr()))
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
		p.Error(errors.Token(errors.ErrExpectedExprAfterOpenRange, op))
	}
	// Rest if no expression on the right: [items...]
	if _, ok := left.(*ast.Discard); ok {
		// _... not allowed
		p.Error(errors.Node(errors.ErrUnderscoreWithRest, left))
	}
	return &ast.RestExpression{Expression: left}
}

func (p *Parser) ParsePipeline(left ast.Expression, bp BindingPower) *ast.PipelineExpression {
	returnIndex := -1
	steps := make([]ast.Node, 1, 2)
	steps[0] = left // First step

	for p.CurrKind() == lexer.Pipeline {
		p.Advance()
		// Return in a pipeline returns the previous result.
		// Return should be the last step, without parameters, and should
		// only be used in expression statements
		if p.CurrKind() == lexer.Return {
			returnIndex = len(steps)
			steps = append(steps, p.ParseStatement())
		}
		steps = append(steps, p.ParseExpression(bp))
	}
	// Return must be the last step. The type checker will also make sure this
	// pipeline is not used as an expression.
	if returnIndex >= 0 && returnIndex != len(steps)-1 {
		p.Error(errors.Node(errors.ErrReturnPipelineNotLast, steps[returnIndex]))
	}
	return &ast.PipelineExpression{Steps: steps}
}

func (p *Parser) ParseRegexLiteral() *ast.RegexLiteral {
	var (
		isEscape bool
		b        strings.Builder
		lastPos  lexer.Position
	)
	r := &ast.RegexLiteral{}
	slashCol := p.Expect(lexer.Slash).Position.Col
	for p.HasTokens() {
		if curr := p.CurrKind(); curr == lexer.Slash && !isEscape {
			break
		} else if p.Curr().Source == `\` {
			// BUG: Regexes can't contain #!, // or /* because comments
			// are pre-parsed (errors are created if unterminated)
			isEscape = !isEscape
		}
		// Including tokens of any kind, including illegal
		tok := p.Advance()
		offset := tok.Col - slashCol - 1

		switch {
		case tok.Source == "\n":
			continue
		case tok.Line == lastPos.Line:
			// Add spaces between tokens
			b.Write(char.Repeat(' ', int(tok.Col-lastPos.Col)))
		case lastPos.Col == 0:
		case offset > 0:
			// Trim whitespace from start of line if aligned with beginning /
			// similar to backtick strings
			b.Write(char.Repeat(' ', int(offset)))
		}
		if !r.Multiline {
			r.Multiline = tok.Line != lastPos.Line
		}
		b.WriteString(tok.Source)
		lastPos = ranges.TokenEnd(tok)
	}
	r.Source = b.String()
	err := errors.Position(errors.ErrUnterminatedRegex, p.Curr().Position)
	endSlashPos := p.ExpectError(err, lexer.Slash).Position
	// Manually add EOS because regex ends in / which is operator
	curr := p.Curr()
	switch {
	case curr.Position.Line > endSlashPos.Line && !ContinuesStatement(curr.Kind),
		curr.Kind == lexer.EOF,
		curr.Kind == lexer.RightCurlyBrace:
		p.Tokens = slices.Insert(
			p.Tokens, p.Index, lexer.Token{Kind: lexer.Newline, Source: "\n"},
		)
	case curr.Kind == lexer.Identifier &&
		ranges.HasOffset(curr.Position, endSlashPos, 0, 1):
		r.Flags = []rune(p.Advance().Source)
	}
	return r
}

func (p *Parser) ParseVersion(left *ast.Symbol, bp BindingPower) ast.Expression {
	var (
		b     strings.Builder
		err   bool
		first = left.Identifier
	)
	expect := func(kind lexer.TokenType) string {
		tok := p.Advance()
		if tok.Kind != kind {
			err = true
		}
		return tok.Source
	}
	// Check first part of version
	if first[0] != 'v' || len(first) < 2 {
		err = true
	} else {
		for _, c := range first[1:] {
			if !lexer.IsDigit(c) {
				err = true
				break
			}
		}
	}
	b.WriteString(first)
	for p.CurrKind() == lexer.Numeric {
		b.WriteString(p.Advance().Source)
	}
	if p.CurrKind() == lexer.Minus {
		b.WriteString(p.Advance().Source)
		b.WriteString(expect(lexer.Identifier))
		if p.CurrKind() == lexer.Minus {
			b.WriteString(p.Advance().Source)
			b.WriteString(expect(lexer.Numeric))
		}
	}
	ver := &ast.VersionLiteral{Version: b.String()}
	ver.SetPos(left.GetRange().Start, p.lastTokEnd())
	if err {
		p.Error(errors.Node(errors.ErrInvalidVersion, ver))
		return &ast.BadExpression{Value: ver}
	}
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
				p.Error(errors.Node(errors.ErrInvalidObjectPipeStep, lhs))
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
				p.Error(errors.Node(errors.ErrInvalidObjectPipeStep, lhs))
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
		p.Error(errors.Token(errors.ErrInvalidForExprOperator, p.Curr()))
		p.AdvanceNonBoundary()
		// TODO: should we still parse an expression after?
	case isAssignment(k), k == lexer.Arrow:
		// -> or any assignment except := or =
		f.Operator = newOperator(p.Advance())
		// Allow spread (...) to be included at the end, to spread entire loop.
		f.Value = p.ParseExpressionWithout(excludeIf(lexer.Ellipsis), RangeBindingPower, try)
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
			p.Error(errors.Node(errors.ErrMustBeFuncCall, g.Expression).
				SetParam("expr", lexer.Go),
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
		p.Error(errors.Token(errors.ErrTryBlock, p.Curr()))
		p.ParseBlock() // Just parse it
		return t
	}
	t.Expression = p.ParseExpression(UnaryBindingPower)
	if _, ok := t.Expression.(*ast.CallExpression); !ok {
		p.Error(errors.Node(errors.ErrMustBeFuncCall, t.Expression).
			SetParam("expr", lexer.Try),
		)
	}
	return t
}

func (p *Parser) ParseAssertExpression(left ast.Expression) *ast.AssertExpression {
	p.Advance() // !!
	return &ast.AssertExpression{Expression: left}
}
