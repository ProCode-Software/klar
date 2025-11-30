package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) ParseVarTypeAnnotation(left ast.Node, bp BindingPower) ast.Statement {
	p.Advance() // :
	var v *ast.DestructureVars
	switch left := left.(type) {
	case *ast.DestructureVars:
		v = left
	case *ast.Symbol, *ast.Discard, *ast.BadExpression:
		v = &ast.DestructureVars{
			Values: []ast.Assignable{left.(ast.Assignable)},
		}
	default:
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
		v = &ast.DestructureVars{
			Values: []ast.Assignable{&ast.BadExpression{Value: left}},
		}
	}
	annot := &ast.TypeAnnotation{
		Variable: v,
		Type:     p.ParseType(DefaultTypeBindingPower),
	}
	switch curr := p.Curr(); curr.Kind {
	case lexer.Equal, lexer.PlusEqual, lexer.MinusEqual:
		p.Error(errors.Node(errors.ErrInvalidTypeAnnotation, annot))
		fallthrough
	case lexer.ColonEqual:
		return p.ParseAssignment(v, bpOf(curr.Kind))
	default:
		p.Error(errors.ExpectedToken(lexer.ColonEqual, curr))
		return &ast.BadExpression{Value: annot}
	}
}

func (p *Parser) ParseVariableDeclaration(left ast.Expression, right []ast.Expression) *ast.VariableDeclaration {
	var explicitType ast.Type
	var vars []ast.Destructure
	if annot, ok := left.(*ast.TypeAnnotation); ok {
		left, explicitType = annot.Variable, annot.Type
	}
	switch left := left.(type) {
	case *ast.DestructureVars:
		vars = make([]ast.Destructure, len(left.Values))
		for i, v := range left.Values {
			if _, ok := v.(ast.Destructure); !ok {
				p.Error(errors.Node(errors.ErrNonNameDeclaration, v))
				v = &ast.BadExpression{Value: v}
			}
			vars[i] = v.(ast.Destructure)
		}
	case *ast.Symbol, *ast.Discard:
		vars = []ast.Destructure{left.(ast.Destructure)}
	default:
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
		vars = []ast.Destructure{&ast.BadExpression{Value: left}}
	}
	return &ast.VariableDeclaration{
		Variables:    vars,
		Values:       right,
		ExplicitType: explicitType,
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(left ast.Expression, bp BindingPower) ast.Statement {
	op := p.Advance()
	var values []ast.Assignable
	switch left := left.(type) {
	case *ast.DestructureVars:
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
		rhs = append(rhs, p.ParseExpression(bp))
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

func (p *Parser) ParseCommaStatement(first ast.Expression, bp BindingPower) ast.Statement {
	items := make([]ast.Assignable, 1, 2)
	items[0] = p.validateAssignableOrFix(first)
	for p.CurrKind() == lexer.Comma {
		p.Advance()
		items = append(items, p.ParseDestructure())
	}
	d := &ast.DestructureVars{Values: items}
	if curr := p.CurrKind(); isAssignment(curr) {
		return p.ParseAssignment(d, bpOf(curr))
	} else if curr == lexer.Colon {
		return p.ParseVarTypeAnnotation(d, bpOf(curr))
	}
	p.Error(errors.Slice(errors.ErrInvalidComma, items))
	return &ast.BadExpression{Value: d}
}

// Soft keywords are not allowed in module names
func (p *Parser) ParseImportStatement() *ast.ImportStatement {
	i := &ast.ImportStatement{}
	p.Advance() // import
	if p.CurrKind() == lexer.EndOfStatement {
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
		i.Alias = p.ParseIdentifier()
		p.Advance() // =
	}
	// First part of module
	i.Module = append(i.Module, p.ParseStrictIdentifier())
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
			if !i.Alias.IsZero() {
				p.Error(errors.Token(errors.ErrWildcardAndAlias, wc))
			}
			break
		} else if curr == lexer.LeftCurlyBrace {
			break
		}
		i.Module = append(i.Module, p.ParseStrictIdentifier())
	}
unqualifiedImport:
	// Unqualified import
	if p.CurrKind() == lexer.LeftCurlyBrace {
		p.Advance() // {
		switch {
		case !i.Alias.IsZero():
			// Alias and unqualified import
			p.Error(errors.Token(errors.ErrAliasInUnqualifiedImport, p.PeekBehind()))
		case i.Wildcard:
			// Wildcard and unqualified import //nolint:dupword
			// import module.*.{...}
			p.Error(errors.Token(errors.ErrWildcardWithUnqualified, p.PeekBehind()))
		case p.CurrKind() == lexer.RightCurlyBrace:
			p.Error(errors.Token(errors.ErrEmptyUnqualifiedImport, p.Curr()))
		}
		parseSeries(p, &i.UnqualifiedImports, func() (u *ast.UnqualifiedImport) {
			u = &ast.UnqualifiedImport{}
			if p.PeekKind() == lexer.Colon {
				u.Alias = p.ParseIdentOrDiscard()
				p.Advance() // :
			}
			u.Identifier = p.ParseIdentifier()
			return
		}, lexer.RightCurlyBrace, lexer.Comma, true)
	}

	return i
}

func (p *Parser) ParseReturnStatement() *ast.ReturnStatement {
	p.Expect(lexer.Return)
	if p.CurrKind() == lexer.EndOfStatement {
		return &ast.ReturnStatement{}
	}
	return &ast.ReturnStatement{
		Value: p.ParseExpression(DefaultBindingPower),
	}
}

func (p *Parser) ParseUpdateStatement(left ast.Node) *ast.UpdateStatement {
	op := p.Expect(lexer.PlusPlus, lexer.MinusMinus)
	var l ast.Expression
	switch left.(type) {
	case *ast.Symbol, *ast.IndexExpression:
		l = left.(ast.Expression)
	default:
		l = &ast.BadExpression{Value: left}
		p.Error(errors.Node(errors.ErrInvalidUpdateExpr, left))
	}
	return &ast.UpdateStatement{Operator: newOperator(op), Left: l}
}

func (p *Parser) ParseForStatement() *ast.ForStatement {
	p.Expect(lexer.For)
	f := &ast.ForStatement{}
	if p.CurrKind() == lexer.LeftCurlyBrace {
		p.Error(errors.Token(errors.ErrForInvalidCondition, p.Curr()))
		goto body
	}
	// Peek for `in` before parsing destructure
	if k := p.PeekKind(); p.IsAssignmentStart() || k == lexer.Comma ||
		k == lexer.Colon || k == lexer.In {
		f.Variables = p.ParseDestructureTypePairs(false)
		p.Expect(lexer.In)
	}
	f.Expression = p.ParseExpression(ExpressionBindingPower)
body:
	f.Body = p.ParseBlock()
	return f
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
	start := p.Expect(lexer.LeftCurlyBrace).Position
	for p.WhileNot(lexer.RightCurlyBrace) {
		body = append(body, p.ParseStatement())
	}
	end := p.Expect(lexer.RightCurlyBrace).Position
	return &ast.Block{
		Body:     body,
		BaseNode: ast.BaseNode{Range: ranges.Range{start, end}},
	}
}

func (p *Parser) ParseControlStatement() ast.Statement {
	stmtKind := p.Advance().Kind // next, stop
	var loopKind lexer.TokenType
	switch p.CurrKind() {
	case lexer.EndOfStatement:
	case lexer.When, lexer.For, lexer.While:
		p.Advance()
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
