package parser

import (
	"strings"

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
	case *ast.Symbol:
		v = &ast.DestructureVars{Values: []ast.Assignable{left}}
	default:
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
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

func (p *Parser) ParseVariableDeclaration(left, right ast.Expression) *ast.VariableDeclaration {
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
	case *ast.Symbol:
		vars = []ast.Destructure{left}
	default:
		p.Error(errors.Node(errors.ErrNonNameDeclaration, left))
	}
	return &ast.VariableDeclaration{
		Variables:    vars,
		Value:        right,
		ExplicitType: explicitType,
	}
}

// ParseAssignment parses a variable declaration or reassignment statement.
func (p *Parser) ParseAssignment(left ast.Expression, bp BindingPower) ast.Statement {
	op := p.Advance()
	rhs := p.ParseExpression(bp)
	if op.Kind == lexer.ColonEqual {
		return p.ParseVariableDeclaration(left, rhs)
	}
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
	return &ast.AssignmentStatement{
		Assignee: values,
		Operator: newOperator(op),
		Value:    rhs,
	}
}

// Soft keywords are not allowed in module names
func (p *Parser) ParseImportStatement() *ast.ImportStatement {
	var b strings.Builder
	i := &ast.ImportStatement{}
	p.Advance() // import
	// Parse maybe alias
	if p.isEqualOrColonEqualAndError(p.Peek()) {
		i.Alias = p.ParseIdentifier()
		p.Advance() // =
	}
	// First part of module
	first := p.Expect(lexer.Identifier)
	modStart := first.Position
	b.WriteString(first.Source)
	for p.CurrKind() == lexer.Dot {
		p.Advance()
		// Wildcard import
		if curr := p.CurrKind(); curr == lexer.Asterisk {
			i.Wildcard = true
			p.Advance()
			break
		} else if curr == lexer.LeftCurlyBrace {
			break
		}
		b.WriteByte('.')
		b.WriteString(p.Expect(lexer.Identifier).Source)
	}
	i.Module = ast.Identifier{Position: modStart, Name: b.String()}

	// Unqualified import
	if p.CurrKind() == lexer.LeftCurlyBrace {
		p.Advance() // {
		switch {
		case i.Alias.Name != "":
			// Alias and unqualified import
			p.Error(errors.Token(errors.ErrAliasInUnqualifiedImport, p.PeekBehind()))
		case i.Wildcard:
			// Wildcard and unqualified import
			// import module.*.{...}
			p.Error(errors.Token(errors.ErrWildcardAndUnqImport, p.PeekBehind()))
		case p.CurrKind() == lexer.RightCurlyBrace:
			p.Error(errors.Token(errors.ErrEmptyUnqImport, p.Curr()))
		}
		parseSeries(p, &i.UnqualifiedImports, func() (u *ast.UnqualifiedImport) {
			u = &ast.UnqualifiedImport{}
			if p.CurrKind() == lexer.Asterisk {
				p.Advance()
				u.Wildcard = true
				return
			}
			if p.Peek().Kind == lexer.Colon {
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

func (p *Parser) ParseUpdateStatement(left ast.Expression) *ast.UpdateStatement {
	op := p.Expect(lexer.PlusPlus, lexer.MinusMinus)
	return &ast.UpdateStatement{Operator: newOperator(op), Left: left}
}

func (p *Parser) ParseForStatement() *ast.ForStatement {
	p.Expect(lexer.For)
	f := &ast.ForStatement{}
	// Peek for `in` before parsing destructure
	if p.IsAssignmentStart() {
		f.Variables = p.ParseDestructureTypePairs(false)
		p.Expect(lexer.In)
	}
	f.Expression = p.ParseExpression(ExpressionBindingPower)
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
