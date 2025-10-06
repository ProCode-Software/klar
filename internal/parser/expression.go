package parser

import (
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseBinaryExpression(left ast.Expression, bp BindingPower) *ast.BinaryExpression {
	op := p.Advance()
	right := p.ParseExpression(bp)
	return &ast.BinaryExpression{
		Left:     left,
		Operator: newOperator(op),
		Right:    right,
	}
}

func (p *Parser) ParseUnaryExpression() *ast.UnaryExpression {
	op := p.Advance()
	right := p.ParseExpression(UnaryBindingPower)
	return &ast.UnaryExpression{Operator: newOperator(op), Right: right}
}

// TODO: fix funcs such as ParseRange for checking if left is ast.Expression
func (p *Parser) ParseRelationalExpression(left ast.Expression, bp BindingPower) *ast.RelationalExpression {
	rel := &ast.RelationalExpression{}
	rel.Expressions = append(rel.Expressions, left) // First expression
	for isRelational(p.CurrKind()) {
		rel.Operators = append(rel.Operators, newOperator(p.Advance()))
		rel.Expressions = append(rel.Expressions, p.ParseExpression(bp))
	}
	return rel
}

func (p *Parser) ParseParamList() *ast.DestructureTuple {
	p.Expect(lexer.LeftParenthesis)
	tuple := &ast.DestructureTuple{}
	if p.CurrKind() != lexer.RightParenthesis {
		tuple.Values = p.ParseDestructureTypePairs(true)
	}
	p.Expect(lexer.RightParenthesis)
	return tuple
}

func (p *Parser) ParseParenExpression() ast.Expression {
	if p.IsArrowFuncStart() {
		return p.ParseParamList()
	}
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
			p.Expect(lexer.EndOfStatement, lexer.Comma)
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
	var (
		leftExpr, rightExpr ast.Expression
		isSlice             bool
	)
	// Slice [:3]
	if p.CurrKind() == lexer.Colon {
		isSlice = true
		p.Advance()
		if p.CurrKind() == lexer.RightBracket {
			// Slice all [:]
			p.Advance()
			return &ast.SliceExpression{Object: leftExpr}
		}
	}
	// Expression
	item = p.ParseExpression(ExpressionBindingPower)

	if isSlice {
		rightExpr = item
	} else if p.CurrKind() == lexer.Colon {
		isSlice = true
		leftExpr = item
		p.Advance()
		// Slice [1:]
		if p.CurrKind() != lexer.RightBracket {
			rightExpr = p.ParseExpression(ExpressionBindingPower)
		}
	}
	p.Expect(lexer.RightBracket)

	if isSlice {
		return &ast.SliceExpression{
			Object: left,
			Start:  leftExpr,
			High:   rightExpr,
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
			arg.Label, arg.Value = symbolToIdentifier(key), val
		case p.Peek().Kind == lexer.Colon:
			// Label (allow keywords)
			arg.Label = p.ParseMapIdentifier(isLabel)
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

func (p *Parser) ParseLambda(left ast.Expression, bp BindingPower) *ast.LambdaExpression {
	l := &ast.LambdaExpression{}
	p.Expect(lexer.Arrow)
	// TODO: destructure for [...] and #{...}
	switch left := left.(type) {
	case *ast.Symbol, *ast.Discard:
		l.Params = []*ast.DestructureTypePair{
			{Keys: []ast.Destructure{left.(ast.Destructure)}},
		}
	case *ast.DestructureTuple:
		l.Params = left.Values
	default:
		p.Error(errors.Node(errors.ErrInvalidLambdaParams, left))
	}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		l.Body = p.ParseBlock()
	} else {
		l.ExprBody = p.ParseExpression(ExpressionBindingPower)
	}
	return l
}

func (p *Parser) ParseLeftRest() *ast.RestExpression {
	p.Expect(lexer.Ellipsis)
	// Allow [...] in when case
	// But not ..._
	if p.CurrKind() == lexer.Underscore {
		p.Error(errors.Token(errors.ErrUnderscoreWithRest, p.Curr()))
	}
	if p.isWhenCase {
		switch p.CurrKind() {
		case lexer.Comma, lexer.RightBracket,
			lexer.RightParenthesis, lexer.RightCurlyBrace:
			return &ast.RestExpression{Left: true}
		}
	}
	return &ast.RestExpression{
		Left:       true,
		Expression: p.ParseExpression(UnaryBindingPower),
	}
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
			p.Error(errors.Token(errors.ErrEllipsisForClosedRange, p.Curr()))
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
		p.Error(errors.Token(errors.ErrExpectedExprAfterClosedRange, op))
	}
	// Rest if no expression on the right: [items...]
	if _, ok := left.(*ast.Discard); ok {
		// _... not allowed
		p.Error(errors.Node(errors.ErrUnderscoreWithRest, left))
	}
	return &ast.RestExpression{Left: false, Expression: left}
}

func (p *Parser) ParsePipeline(left ast.Expression, bp BindingPower) *ast.PipelineExpression {
	var returnIndex int
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
	if returnIndex != len(steps)-1 {
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
		} else if curr == lexer.Backslash {
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
			b.Write(repeatByte(' ', int(tok.Col-lastPos.Col)))
		case lastPos.Col == 0:
		case offset > 0:
			// Trim whitespace from start of line if aligned with beginning /
			// similar to backtick strings
			b.Write(repeatByte(' ', int(offset)))
		}
		if !r.Multiline {
			r.Multiline = tok.Line != lastPos.Line
		}
		b.WriteString(tok.Source)
		lastPos = ranges.FromToken(tok).End
	}
	r.Source = b.String()
	err := errors.Position(errors.ErrUnterminatedRegex, p.Curr().Position)
	endSlashPos := p.ExpectError(err, lexer.Slash).Position
	// Manually add EOS because regex ends in / which is operator
	curr := p.Curr()
	switch {
	case curr.Position.Line > endSlashPos.Line && !CanGoOnNewline(curr.Kind),
		curr.Kind == lexer.EOF,
		curr.Kind == lexer.RightCurlyBrace:
		p.Tokens = slices.Insert(
			p.Tokens, p.Index, lexer.Token{Kind: lexer.EndOfStatement, Source: "\n"},
		)
	case curr.Kind == lexer.Identifier &&
		curr.Position == ranges.Add(endSlashPos, 0, 1):
		r.Flags = []rune(p.Advance().Source)
	}
	return r
}

func (p *Parser) ParseVersion(left ast.Expression, bp BindingPower) ast.Expression {
	var (
		b     strings.Builder
		err   bool
		first = left.(*ast.Symbol).Identifier
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
		p.Error(errors.Node(errors.ErrInvalidVersionLit, ver))
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
				p.Error(errors.Node(errors.ErrInvalidObjPipeStep, lhs))
				lhs = &ast.BadExpression{Value: lhs}
			}
		}
		// Index or call
		if k := p.CurrKind(); !isAssignment(k) && k != lexer.StrokeDot {
			lhs = p.ParseLED(lhs, bp)
		}
		// Assignment
		if k := p.CurrKind(); isAssignment(k) && k != lexer.ColonEqual {
			l := p.validateAssignableOrFix(lhs)
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
				p.Error(errors.Node(errors.ErrInvalidObjPipeStep, lhs))
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
	// Peek for `in` before parsing destructure
	if p.IsAssignmentStart() {
		f.Variables = p.ParseDestructureTypePairs(false)
		p.Expect(lexer.In)
	}
	f.Iterator = p.ParseExpression(LambdaBindingPower)
	p.isEqualOrColonEqualAndError() // Report error if ':='
	switch p.CurrKind() {
	case lexer.Equal, lexer.PlusEqual, lexer.MinusEqual, lexer.Arrow:
		f.Operator = newOperator(p.Advance())
		f.Value = p.ParseExpression(ExpressionBindingPower)
	case lexer.LeftCurlyBrace:
		f.Block = p.ParseBlock()
	}
	return f
}

func (p *Parser) ParseGoExpression() *ast.GoExpression {
	p.Advance() // go
	g := &ast.GoExpression{}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		g.Body = p.ParseBlock()
	} else {
		g.Expression = p.ParseExpression(UnaryBindingPower)
		if _, ok := g.Expression.(*ast.CallExpression); !ok {
			p.Error(errors.Node(errors.ErrMustBeFuncCall, g.Expression))
		}
	}
	return g
}

func (p *Parser) ParseAwaitExpression() *ast.AwaitExpression {
	p.Advance() // await
	return &ast.AwaitExpression{Expression: p.ParseExpression(UnaryBindingPower)}
}

func (p *Parser) ParseTryExpression() *ast.TryExpression {
	p.Advance() // try
	t := &ast.TryExpression{}
	t.Expression = p.ParseExpression(UnaryBindingPower)
	if _, ok := t.Expression.(*ast.CallExpression); !ok {
		p.Error(errors.Node(errors.ErrMustBeFuncCall, t.Expression))
	}
	return t
}

func (p *Parser) ParseTernaryExpression(left ast.Expression, bp BindingPower) ast.Expression {
	p.Advance() // if
	t := &ast.TernaryExpression{Value: left}
	t.Condition = p.ParseExpression(bp)
	p.Expect(lexer.Else)
	t.Else = p.ParseExpression(bp)
	return t
}
