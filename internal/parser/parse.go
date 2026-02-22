package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Parse parses p.Tokens into a [*ast.Program].
func (p *Parser) Parse() *ast.Program {
	defer p.handlePanic()
	var (
		body     = make([]ast.Statement, 0, len(p.Tokens)/10)
		comments = p.InsertEOS()
	)
	for p.HasTokens() {
		if p.CurrKind() == lexer.Newline {
			if p.Advance(); !p.HasTokens() {
				break
			}
		}
		body = append(body, p.ParseTopLevelStatement())
	}
	return &ast.Program{
		Body:     slices.Clip(body),
		Comments: comments,
		BaseNode: ast.BaseNode{Range: ranges.Range{
			Start: p.Tokens[0].Position,
			End:   p.Tokens[len(p.Tokens)-1].Position,
		}},
	}
}

func (p *Parser) ParseStatement(flags uint8) ast.Statement {
	kind := p.CurrKind()
	expectEOS := func() {
		if (flags & allowCommaTerminator) != 0 {
			p.Expect(lexer.Comma, lexer.Newline)
		} else if (flags & noEOS) == 0 {
			p.Expect(lexer.Newline)
		}
	}
	// A. Try to parse a full statement
	if res, handled := p.handleStatement(kind); handled {
		expectEOS()
		return res
	}
	var expr ast.Expression
	var ok bool
	// B. Start with a NUD
	if next := p.PeekKind(); kind == lexer.Underscore &&
		(isAssignment(next) || next == lexer.Comma || next == lexer.Colon) {
		// Allow discard assignments
		p.Advance()
		expr = &ast.Discard{}
	} else if expr, ok = p.handleNUD(kind); !ok { // Expression NUD
		p.nudError()
		// expectEOS() // TODO: should we?
		return &ast.BadExpression{Token: kind}
	}

	// C. Try to parse a statement LED
	if stmt, ok := p.handleStatementLED(p.CurrKind(), expr); ok {
		expectEOS()
		return stmt
	}
	// D. Otherwise parse an expression LED
	expr = p.ParseLED(expr, ExpressionBindingPower)

	// E. Then parse a statement LED after the expression, unless a comma
	// is a terminator (don't parse comma assignments)
	if flags&allowCommaTerminator == 0 || p.CurrKind() != lexer.Comma {
		if stmt, ok := p.handleStatementLED(p.CurrKind(), expr); ok {
			// Assignment statement
			expectEOS()
			return stmt
		}
	}
	// F. This statement is an expression statement
	expectEOS()
	return copyPos(expr, &ast.ExpressionStatement{Expression: expr})
}

func (p *Parser) ParseTopLevelStatement() ast.Statement {
	kind := p.CurrKind()
	res, handled := p.handleTopLevelStatement(kind)
	if handled {
		if pb := p.PeekBehind(); pb.Kind != lexer.Asterisk {
			p.Expect(lexer.Newline)
		} else if c := p.Curr(); c.Position.Line == pb.Position.Line &&
			c.Kind != lexer.Newline && c.Kind != lexer.EOF {
			// No newline after import
			p.Error(errors.ExpectedToken(lexer.Newline, c))
		}
		return res
	}
	return p.ParseStatement(0)
}

// ParseComment takes tok, a [lexer.LineComment], [lexer.BlockComment] or [lexer.Hashbang]
// token and parses it into an [*ast.Comment] node.
// Errors are reported to the parser if block comments are unterminated or shebangs
// are not on the first line. These are the first errors reported in the parsing process.
func (p *Parser) ParseComment(tok lexer.Token) *ast.Comment {
	end := len(tok.Source)
	switch {
	case tok.Kind == lexer.Hashbang:
		if tok.Position != (lexer.Position{1, 1}) {
			p.Error(errors.Token(errors.ErrMisplacedShebang, tok))
		}
		// TODO: maybe error hints for newlines/spaces before
	case tok.Attributes["unterm"] == true:
		p.Error(&errors.ParseError{
			ErrorCode: errors.ErrUnterminatedComment,
			Token:     tok,
			Range:     ranges.Offset(tok.Position, 0, 1),
		})
	case tok.Kind == lexer.BlockComment: // But not if unterminated
		end -= 2
	}
	return &ast.Comment{
		Value:    tok.Source[2:end],
		Type:     tok.Kind,
		BaseNode: ast.BaseNode{ranges.FromToken(tok)},
	}
}

func (p *Parser) ParseExpression(bp BindingPower) ast.Expression {
	return p.ParseFull(bp)
}

func (p *Parser) ParseFull(bp BindingPower) ast.Expression {
	kind := p.CurrKind()
	if kind == lexer.EOF {
		return &ast.BadExpression{Token: kind}
	}
	left, handled := p.handleNUD(kind)
	if !handled {
		p.nudError()
		return &ast.BadExpression{Token: kind}
	}
	return p.ParseLED(left, bp)
}

func (p *Parser) TryParseLED(left ast.Expression, bp BindingPower) (ast.Expression, bool) {
	kind := p.CurrKind()
	left, handled := p.handleLED(kind, left, BindingPowerMap[kind])
	if !handled {
		return left, false
	}
	return p.ParseLED(left, bp), true
}

func (p *Parser) ParseLED(left ast.Expression, bp BindingPower) ast.Expression {
	var handled bool
	for BindingPowerMap[p.CurrKind()] > bp {
		kind := p.CurrKind()
		left, handled = p.handleLED(kind, left, BindingPowerMap[kind])
		if !handled {
			p.unknownTokenError()
			return &ast.BadExpression{Token: kind, Value: left}
		}
	}
	// left = left.SetPos(left.GetRange().Start, p.savePos())
	return left
}

// ParseExpressionWithout parses an expression, stopping if exclude returns
// true for the current token type.
func (p *Parser) ParseExpressionWithout(
	exclude func(lexer.TokenType) bool, initialBP BindingPower, flags uint8,
) ast.Expression {
	expr := p.ParseExpression(initialBP)
	if exclude(p.CurrKind()) &&
		(flags&allowIfSameLine == 0 || p.Curr().Line != expr.GetRange().End.Line) {
		return expr
	}
	if (flags & try) != 0 {
		expr, _ = p.TryParseLED(expr, ExpressionBindingPower)
		return expr
	}
	return p.ParseLED(expr, ExpressionBindingPower)
}

func excludeIf(kind lexer.TokenType) func(lexer.TokenType) bool {
	return func(t lexer.TokenType) bool { return t == kind }
}

func parseFlags(flgs []uint8) uint8 {
	var flags uint8
	for _, flag := range flgs {
		flags |= flag
	}
	return flags
}

const (
	noEOS uint8 = 1 << iota
	allowCommaTerminator
	try
	allowIfSameLine
	allowNumber
	isLabel
)

func parseExprSeries[T ast.Node](
	p *Parser, arr *[]T,
	bp BindingPower, until, sepBy lexer.TokenType,
) {
	parseSeries(
		p, arr,
		func() T { return p.ParseExpression(bp).(T) },
		until, sepBy, false,
	)
}

func parseSeries[T ast.Node](
	p *Parser, arr *[]T,
	with func() T, until, separator lexer.TokenType,
	allowEOS bool,
) {
	allSeps := make([]lexer.TokenType, 1, 2)
	allSeps[0] = separator
	if allowEOS {
		allSeps = append(allSeps, lexer.Newline)
	}
	if until == 0 && separator == 0 {
		panic("until and separator cannot both be zero")
	}
	for p.HasTokens() && p.CurrKind() != until {
		if !allowEOS && p.CurrKind() == lexer.Newline {
			break
		}
		start := p.Curr().Position
		*arr = append(*arr, markStartEndPos(p, with(), start))
		if !p.HasTokens() {
			break
		}
		curr := p.CurrKind()
		if curr == until || (until == 0 && curr != separator) ||
			(!allowEOS && curr == lexer.Newline) {
			break
		}
		if separator != 0 {
			p.Expect(allSeps...)
		}
	}
	if until != 0 {
		p.Expect(until)
	}
}
