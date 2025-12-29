package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Parse parses p.Tokens into a [*ast.Program].
func (p *Parser) Parse() *ast.Program {
	defer p.handlePanic()
	var (
		body     = make([]ast.Statement, 0, len(p.Tokens)/3)
		comments = p.InsertEOS()
	)
	for p.HasTokens() {
		if p.CurrKind() == lexer.EndOfStatement {
			if p.Advance(); !p.HasTokens() {
				break
			}
		}
		body = append(body, p.ParseTopLevelStatement())
	}
	return &ast.Program{
		Body:     body[:len(body):len(body)], // Remove unused length
		Comments: comments,
		BaseNode: ast.BaseNode{Range: ranges.Range{
			Start: p.Tokens[0].Position,
			End:   p.Tokens[len(p.Tokens)-1].Position,
		}},
	}
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

func (p *Parser) unknownTokenErr() {
	p.Error(errors.UnexpectedToken(p.AdvanceNonBoundary()))
	p.skipUntilBoundary()
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

func (p *Parser) nudError() {
	switch curr := p.Curr(); curr.Kind {
	case lexer.Illegal:
		if p.checkIllegal(curr) {
			break
		}
		fallthrough
	default:
		p.unknownTokenErr()
		return
	case lexer.If:
		p.Error(errors.Token(errors.ErrIfStatement, curr))
	}
	p.skipUntilBoundary()
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
			p.unknownTokenErr()
			return &ast.BadExpression{Token: kind, Value: left}
		}
	}
	// left = left.SetPos(left.GetRange().Start, p.savePos())
	return left
}

func (p *Parser) ParseTopLevelStatement() ast.Statement {
	kind := p.CurrKind()
	res, handled := p.handleTopLevelStatement(kind)
	if handled {
		if pb := p.PeekBehind(); pb.Kind != lexer.Asterisk {
			p.Expect(lexer.EndOfStatement)
		} else if c := p.Curr(); c.Position.Line == pb.Position.Line &&
			c.Kind != lexer.EndOfStatement && c.Kind != lexer.EOF {
			// No newline after import
			p.Error(errors.ExpectedToken(lexer.EndOfStatement, c))
		}
		return res
	}
	return p.ParseStatement()
}

func parseFlags(flgs []int) int {
	if len(flgs) == 0 {
		return 0
	} else if len(flgs) == 1 {
		return flgs[0]
	}
	var flags int
	for _, flag := range flgs {
		flags |= flag
	}
	return flags
}

const (
	withoutEOS = 1 << iota
	usingComma
)

func (p *Parser) ParseStatement(flags ...int) ast.Statement {
	flag := parseFlags(flags)
	noEOS := (flag & withoutEOS) != 0
	kind := p.CurrKind()
	if res, handled := p.handleStatement(kind); handled {
		if !noEOS {
			p.Expect(lexer.EndOfStatement)
		}
		return res
	}
	var r ast.Node
	res, handled := p.handleStatementNUD(kind)
	if !handled {
		if res, handled = p.handleNUD(kind); !handled {
			p.nudError()
			res = &ast.BadExpression{Token: kind}
		}
	}
	if handled { // reassigned in handleNUD() call
		kind = p.CurrKind()
		comma := noEOS && (flag&usingComma) != 0
		if kind == lexer.Comma && comma {
			goto checkEOS
		}
		r, handled = p.handleStatementLED(kind, res, DefaultBindingPower)
		if !handled {
			r = p.ParseLED(res, AssignBindingPower)
			if comma && p.CurrKind() == lexer.Comma {
				goto checkEOS
			}
			r, _ = p.handleStatementLED(p.CurrKind(), r.(ast.Expression), DefaultBindingPower)
		}
	}
checkEOS:
	if r == nil {
		r = res
	}
	if !noEOS {
		p.Expect(lexer.EndOfStatement)
	}
	switch r := r.(type) {
	// Left-denoted statement
	case ast.Statement:
		return r
	// Then it is an expression
	case ast.Expression:
		stmt := &ast.ExpressionStatement{Expression: r}
		copyPos(r, stmt)
		return stmt
	// I don't know what this is. If this occurs, then it is a bug.
	default:
		panic(fmt.Sprintf("node %v (type %[1]T) is neither an expression nor statement", r))
	}
}

func (p *Parser) skipUntilBoundary() {
	brackCount := 1
	for p.HasTokens() {
		switch p.CurrKind() {
		case lexer.Comma, lexer.EndOfStatement:
			if brackCount <= 1 {
				return
			}
		case lexer.LeftParenthesis, lexer.LeftBracket, lexer.LeftCurlyBrace, lexer.HashLeftCurlyBrace:
			brackCount++
		case lexer.RightParenthesis, lexer.RightBracket, lexer.RightCurlyBrace:
			brackCount--
			if brackCount <= 0 {
				return
			}
		}
		p.Advance()
	}
}

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
		allSeps = append(allSeps, lexer.EndOfStatement)
	}
	if until == 0 && separator == 0 {
		panic("until and separator cannot both be zero")
	}
	for p.HasTokens() && p.CurrKind() != until {
		if !allowEOS && p.CurrKind() == lexer.EndOfStatement {
			break
		}
		start := p.Curr().Position
		*arr = append(*arr, markStartEndPos(p, with(), start))
		if !p.HasTokens() {
			break
		}
		curr := p.CurrKind()
		if curr == until || (until == 0 && curr != separator) ||
			(!allowEOS && curr == lexer.EndOfStatement) {
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
