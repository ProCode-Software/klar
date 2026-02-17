package parser

import (
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) countConsecutiveNot() (n int) {
	for p.HasTokens() {
		switch p.Curr().Kind {
		case lexer.Not:
			n++
		case lexer.NotNot:
			n += 2
		default:
			return
		}
		p.Advance()
	}
	return
}

func (p *Parser) unknownTokenError() {
	p.Error(errors.UnexpectedToken(p.AdvanceNonBoundary()))
	p.skipUntilBoundary()
}

func (p *Parser) nudError() {
	switch curr := p.Curr(); curr.Kind {
	case lexer.Illegal:
		if p.checkIllegal(curr) {
			break
		}
		fallthrough
	default:
		p.unknownTokenError()
		return
	case lexer.If:
		p.Error(errors.Token(errors.ErrIfStatement, curr))
	case lexer.Plus:
		p.Error(errors.Token(errors.ErrPositiveSign, curr))
	case lexer.NotNot:
		count := p.countConsecutiveNot()
		err := errors.Range(errors.ErrDoubleNot,
			ranges.Offset(curr.Position, 0, uint32(count)),
		)
		err.SetParam("count", count)
		p.Error(err)
	}
	p.skipUntilBoundary()
}

func (p *Parser) skipUntilBoundary() {
	brackCount := 1
	for p.HasTokens() {
		switch p.CurrKind() {
		case lexer.Comma, lexer.Newline:
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