package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseVarTypeAnnotation(left ast.Node) ast.Statement {
	p.Advance() // :
	var v *ast.AssignableVars
	switch left := left.(type) {
	case *ast.AssignableVars:
		v = left
	case ast.Assignable:
		v = &ast.AssignableVars{
			Values: []ast.Assignable{left},
		}
	default:
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
		v = &ast.AssignableVars{
			Values: []ast.Assignable{
				p.validateAssignable(&ast.BadExpression{Value: left}),
			},
		}
	}
	annot := &ast.TypeAnnotation{
		Variable: v,
		Type:     p.ParseType(DefaultTypeBindingPower),
	}
	switch curr := p.Curr(); curr.Kind {
	default:
		if !isAssignment(curr.Kind) {
			p.Error(errors.ExpectedToken(lexer.ColonEqual, curr))
			return &ast.BadExpression{Value: annot}
		}
		p.Error(errors.Node(errors.ErrInvalidTypeAnnotation, annot))
		fallthrough
	case lexer.Equal, lexer.PlusEqual, lexer.MinusEqual:
		p.Error(errors.Node(errors.ErrInvalidTypeAnnotation, annot))
		fallthrough
	case lexer.ColonEqual:
		return p.ParseAssignment(v)
	}
}

func (p *Parser) ParseVariableDeclaration(left ast.Expression, right []ast.Expression) *ast.VariableDeclaration {
	var explicitType ast.Type
	var vars []ast.Assignable
	if annot, ok := left.(*ast.TypeAnnotation); ok {
		left, explicitType = annot.Variable, annot.Type
	}
	switch left := left.(type) {
	case *ast.AssignableVars:
		vars = left.Values
	case ast.Assignable:
		vars = []ast.Assignable{left}
	default:
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
		vars = []ast.Assignable{p.validateAssignable(&ast.BadExpression{Value: left})}
	}
	return &ast.VariableDeclaration{
		Variables:    vars,
		Values:       right,
		ExplicitType: explicitType,
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(left ast.Expression) ast.Statement {
	op := p.Advance()
	var values []ast.Assignable
	switch left := left.(type) {
	case *ast.AssignableVars:
		values = left.Values
	case ast.Assignable:
		values = []ast.Assignable{left}
	default:
		p.Error(errors.Node(errors.ErrInvalidAssignment, left))
		values = []ast.Assignable{&ast.BadExpression{Value: left}}
	}
	expLen := len(values)
	rhs := make([]ast.Expression, 0, expLen)
	for p.HasTokens() {
		rhs = append(rhs, p.ParseExpression(ExpressionBindingPower))
		if p.CurrKind() != lexer.Comma {
			break
		}
		p.Advance() // ,
	}
	rhs = slices.Clip(rhs)
	if gotLen := len(rhs); gotLen > 1 && gotLen != expLen {
		err := errors.Slice(errors.ErrMismatchedAssignment, rhs)
		err.Params = errors.ErrorParams{"left": expLen, "right": gotLen}
		p.Error(err)
	}
	if op.Kind == lexer.ColonEqual {
		return p.ParseVariableDeclaration(left, rhs)
	}
	return &ast.AssignmentStatement{
		Assignee: values,
		Operator: newOperator(op),
		Values:   rhs,
	}
}

func (p *Parser) ParseCommaStatement(first ast.Expression) ast.Statement {
	items := make([]ast.Assignable, 1, 2)
	items[0] = p.validateAssignable(first)
	for p.CurrKind() == lexer.Comma {
		p.Advance()
		items = append(items, p.ParseAssignable())
	}
	d := &ast.AssignableVars{Values: items}
	if curr := p.CurrKind(); isAssignment(curr) {
		return p.ParseAssignment(d)
	} else if curr == lexer.Colon {
		return p.ParseVarTypeAnnotation(d)
	}
	p.Error(errors.Slice(errors.ErrInvalidComma, items))
	return &ast.BadExpression{Value: d}
}

// Soft keywords are not allowed in module names
func (p *Parser) ParseImportStatement() *ast.ImportStatement {
	i := &ast.ImportStatement{}
	p.Advance() // import
	if p.CurrKind() == lexer.Newline {
		p.Advance()
	}
	if c := p.CurrKind(); c == lexer.Dot || c == lexer.LeftCurlyBrace {
		p.Error(errors.Token(errors.ErrImportExpectedModule, p.Curr()))
		if c == lexer.Dot {
			p.Advance()
		}
		goto unqualifiedImport
	}
	// Parse maybe alias
	if p.isEqual(p.Peek()) {
		i.Alias = p.ParseIdentOrDiscard()
		p.Advance() // =
	}
	// First part of module
	i.Module = append(i.Module, p.Expect(lexer.Identifier).Source)
	for p.CurrKind() == lexer.Dot {
		p.Advance()
		// Wildcard import
		if curr := p.CurrKind(); curr == lexer.Asterisk {
			i.Wildcard = true
			wc := p.Advance()
			if p.CurrKind() == lexer.Dot {
				p.Error(errors.Token(errors.ErrImportInvalidWildcard, wc))
				continue
			}
			break
		} else if curr == lexer.LeftCurlyBrace {
			break
		}
		i.Module = append(i.Module, p.Expect(lexer.Identifier).Source)
	}
unqualifiedImport:
	// Unqualified import
	if p.CurrKind() == lexer.LeftCurlyBrace {
		p.Advance() // {
		switch {
		case i.Wildcard:
			// Wildcard and unqualified import //nolint:dupword
			// import module.*.{...}
			p.Error(errors.Token(errors.ErrWildcardWithUnqualified, p.PeekBehind()))
		case p.CurrKind() == lexer.RightCurlyBrace:
			p.Error(errors.Token(errors.ErrEmptyUnqualifiedImport, p.Curr()))
		}
		parseSeries(p, &i.UnqualifiedImports, func() (u *ast.IdentifierPair) {
			u = &ast.IdentifierPair{Name: p.ParseIdentifier()}
			// Alias
			if p.CurrKind() == lexer.As {
				p.Advance()
				u.Label = p.ParseIdentifier()
			}
			return
		}, lexer.RightCurlyBrace, lexer.Comma, true)
	}

	return i
}

func (p *Parser) ParseReturnStatement() *ast.ReturnStatement {
	p.Expect(lexer.Return)
	if c := p.CurrKind(); c == lexer.Newline || c == lexer.Comma {
		return &ast.ReturnStatement{}
	}
	return &ast.ReturnStatement{Value: p.ParseExpression(DefaultBindingPower)}
}

func (p *Parser) ParseForStatement() *ast.ForStatement {
	p.Expect(lexer.For)
	f := &ast.ForStatement{}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		p.Error(errors.Token(errors.ErrNoForIterator, p.Curr()))
	} else {
		f.Variables, f.Expression = p.parseForVariables()
	}
	f.Body = p.ParseBlock()
	return f
}

func (p *Parser) parseForVariables() (vars []*ast.AssignableTypePair, iter ast.Expression) {
	first := p.ParseExpression(ExpressionBindingPower)
	if bin, ok := first.(*ast.BinaryExpression); ok && bin.Operator.Kind == lexer.In {
		// for k in m
		// (*AssignableTypePair).Value == always nil
		return []*ast.AssignableTypePair{
			{Keys: []ast.Assignable{p.validateAssignable(bin.Left)}},
		}, bin.Right
	} else if k := p.CurrKind(); k == lexer.Comma || k == lexer.Colon {
		// for k, v in m
		// for k: Int in m
		pair := &ast.AssignableTypePair{
			Keys: []ast.Assignable{p.validateAssignable(first)},
		}
		for p.HasTokens() {
			switch p.CurrKind() {
			case lexer.Comma:
				p.Advance()
				expr := p.ParseExpression(DefaultBindingPower)
				if expr, ok := expr.(*ast.BinaryExpression); ok &&
					expr.Operator.Kind == lexer.In {
					pair.Keys = append(pair.Keys, p.validateAssignable(expr.Left))
					vars = append(vars, pair)
					return vars, expr.Right
				}
				pair.Keys = append(pair.Keys, p.validateAssignable(expr))
			case lexer.Colon:
				p.Advance()
				pair.Type = p.ParseType(DefaultTypeBindingPower)
				vars = append(vars, pair)
				if p.CurrKind() == lexer.In {
					return vars, p.ParseExpression(ExpressionBindingPower)
				}
				pair = &ast.AssignableTypePair{}
			default:
				// TODO
				// p.Error(errors.Token(errors.ErrInvalidForVariables, p.Curr()))
			}
		}
	}
	return nil, first
}

func (p *Parser) ParseWhileStatement() *ast.WhileStatement {
	p.Advance() // while
	w := &ast.WhileStatement{}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		w.Infinite = true
	} else {
		w.Condition = p.ParseExpression(ExpressionBindingPower)
	}
	w.Body = p.ParseBlock()
	return w
}

func (p *Parser) ParseBlock() *ast.Block {
	var body []ast.Statement
	start := p.ExpectNoAdvance(lexer.LeftCurlyBrace).Position
	for p.WhileNot(lexer.RightCurlyBrace) {
		body = append(body, p.ParseStatement())
	}
	end := p.ExpectNoAdvance(lexer.RightCurlyBrace).Position
	return &ast.Block{
		Body:     body,
		BaseNode: ast.BaseNode{Range: ranges.Range{start, end}},
	}
}

func (p *Parser) ParseControlStatement() ast.Statement {
	stmtKind := p.Advance().Kind // next, stop
	var loopKind lexer.TokenType
	switch p.CurrKind() {
	case lexer.Newline, lexer.Comma:
	case lexer.When, lexer.For, lexer.While:
		loopKind = p.Advance().Kind
	default:
		p.Error(errors.Token(errors.ErrInvalidLoop, p.Advance()).
			SetParam("stmt", stmtKind),
		)
	}
	switch stmtKind {
	case lexer.Next:
		return &ast.NextStatement{Loop: loopKind}
	case lexer.Stop:
		return &ast.StopStatement{Loop: loopKind}
	default:
		// Unreachable
		panic("invalid control statement: " + stmtKind.String())
	}
}
