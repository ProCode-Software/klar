package parser

import (
	"fmt"
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseVarTypeAnnotation(left []ast.Assignable) ast.Statement {
	p.Advance() // :
	typ := p.ParseType(DefaultTypeBindingPower)
	switch curr := p.Curr(); curr.Kind {
	default:
		if !isAssignment(curr.Kind) {
			p.Error(errors.ExpectedToken(lexer.ColonEqual, curr))
			return &ast.BadExpression{Value: typ}
		}
		p.Error(errors.Node(errors.ErrInvalidTypeAnnotation, typ))
		fallthrough
	case lexer.Equal, lexer.PlusEqual, lexer.MinusEqual:
		err := errors.Node(errors.ErrInvalidTypeAnnotation, typ)
		if curr.Kind == lexer.Equal {
			err.Hint("Did you mean to use ':='?")
		}
		p.Error(err)
		fallthrough
	case lexer.ColonEqual:
		return p.ParseAssignment(left, typ)
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(lhs []ast.Assignable, explicitType ast.Type) ast.Statement {
	op := p.Advance()
	expLen := len(lhs)
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
		err.Label = fmt.Sprintf("%d values were provided", gotLen)
		plural := "s were"
		if expLen == 1 {
			plural = " was"
		}
		err.Highlights = append(err.Highlights, errors.Highlight{
			Range: ranges.Range{
				Start: lhs[0].GetRange().Start,
				End:   lhs[len(lhs)-1].GetRange().End,
			},
			Message: fmt.Sprintf("%d variable%s provided", expLen, plural),
		})
		err.Params = errors.ErrorParams{"left": expLen, "right": gotLen}
		p.Error(err)
	}
	if op.Kind == lexer.ColonEqual {
		return &ast.VariableDeclaration{
			Variables:    lhs,
			ExplicitType: explicitType,
			Values:       rhs,
		}
	}
	return &ast.AssignmentStatement{
		Assignee: lhs,
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
		return p.ParseAssignment(items, nil)
	} else if curr == lexer.Colon {
		return p.ParseVarTypeAnnotation(items)
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

func (p *Parser) parseForVariables() (
	vars []*ast.AssignableTypePair, iter ast.Expression,
) {
	var first ast.Expression
	if p.CurrKind() != lexer.Underscore {
		first = p.ParseExpressionFilter(excludeIf(lexer.In), bpOf(lexer.In), 0)
		if p.CurrKind() == lexer.LeftCurlyBrace {
			// for 5 {}
			return nil, first
		}
	}
	p.parseAssignableTypePairs(&vars, p.validateAssignable(first), true)
	p.Expect(lexer.In)
	return vars, p.ParseExpression(ExpressionBindingPower)
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
		body = append(body, p.ParseStatement(0))
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
