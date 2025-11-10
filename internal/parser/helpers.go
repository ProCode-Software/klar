package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/printer"
)

func (p *Parser) errExpectedExpr(got ast.Node) {
	err := &errors.ParseError{
		ErrorCode: errors.ErrNotAnExpression,
		Node:      got,
		Range:     got.GetRange(),
	}
	if n, ok := got.(*ast.AssignmentStatement); ok && n.Operator.Kind == lexer.Equal {
		err.Hint("Did you mean to use '==' instead?")
	}
	p.Error(err)
}

func newOperator(t lexer.Token) ast.Operator {
	return ast.Operator{Kind: t.Kind, Position: t.Position}
}

// isEqual returns true if the current token is '='. If it is ':=', isEqual
// reports an error and still returns true.
func (p *Parser) isEqual(token ...lexer.Token) bool {
	t := p.Curr()
	if len(token) > 0 {
		t = token[0]
	}
	switch t.Kind {
	case lexer.ColonEqual:
		p.Error(errors.Token(errors.ErrColonEqual, t))
		return true
	case lexer.Equal:
		return true
	}
	return false
}

func (p *Parser) lastTokEnd() lexer.Position {
	last := p.Tokens[p.Index-1]
	return ranges.FromToken(last).End
}

func (p *Parser) expectShorthand() (key *ast.Symbol, value ast.Expression) {
	var isOk, isComputed bool
	sym := p.ParseExpression(ExpressionBindingPower)
	switch sym := sym.(type) {
	case *ast.Symbol:
		key = sym
		value = sym
		isOk = true
	case *ast.IndexExpression:
		if sym.Computed {
			break
		}
		if prop, ok := sym.Property.(*ast.Symbol); ok {
			key = prop
			value = sym
			isOk = true
		}
	}
	if !isOk {
		err := errors.Node(errors.ErrInvalidLabelShorthand, sym)
		err.Params = errors.ErrorParams{"computed": isComputed}
		p.Error(err)
	}
	return key, value
}

// Range utils
func markEndPos[T ast.Node](p *Parser, node T) T {
	node.SetPos(node.GetRange().Start, p.lastTokEnd())
	return node
}

func markStartEndPos[T ast.Node](p *Parser, node T, start lexer.Position) T {
	node.SetPos(start, p.lastTokEnd())
	return node
}

func rangeFromToken[T ast.Node](node T, tok lexer.Token) T {
	rang := ranges.FromToken(tok)
	node.SetPos(rang.Start, rang.End)
	return node
}

func copyPos[F, T ast.Node](from F, to T) T {
	to.SetPos(from.GetRange().Start, from.GetRange().End)
	return to
}

func newBaseNode(start, end lexer.Position) ast.BaseNode {
	return ast.BaseNode{Range: ranges.Range{start, end}}
}

func isAssignment(kind lexer.TokenType) bool {
	switch kind {
	case lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual:
		return true
	}
	return false
}

func isRelational(kind lexer.TokenType) bool {
	switch kind {
	case lexer.EqualEqual, lexer.NotEqual, lexer.GreaterThan, lexer.LessThan,
		lexer.GreaterEqualTo, lexer.LessEqualTo:
		return true
	}
	return false
}

func printTokens(tok []lexer.Token) []byte {
	return printer.PrintTokens(tok, printer.PrettyPrint|printer.SingleLine)
}

func isComment(t lexer.TokenType) bool {
	switch t {
	case lexer.BlockComment, lexer.LineComment, lexer.Hashbang:
		return true
	}
	return false
}
