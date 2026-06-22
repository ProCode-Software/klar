package parser

import (
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) unexpectedTokenError() {
	tok := p.AdvanceNonBoundary()
	if tok.Kind == lexer.Newline {
		tok = p.Advance()
	}
	p.Error(klarerrs.UnexpectedToken(tok))
	p.skipUntilBoundary()
}

func (p *Parser) skipRestOfStatement() (end lexer.Position) {
	var brackCount int
	for p.HasTokens() {
		tok := p.Advance()
		switch tok.Kind {
		case lexer.Newline:
			if brackCount == 0 {
				return tok.End()
			}
		case lexer.LeftParenthesis, lexer.LeftBracket, lexer.LeftCurlyBrace, lexer.HashLeftCurlyBrace:
			brackCount++
		case lexer.RightParenthesis, lexer.RightBracket, lexer.RightCurlyBrace:
			brackCount--
		}
	}
	return
}

func (p *Parser) nudError() {
	switch curr := p.Curr(); curr.Kind {
	case lexer.Illegal:
		if p.checkIllegal(curr) {
			break
		}
		fallthrough
	default:
		p.unexpectedTokenError()
		return
	case lexer.If:
		p.ErrorLabelled(klarerrs.Token(klarerrs.ErrIfStatement, curr), "Use a 'when' block instead")
	case lexer.Plus:
		p.ErrorLabelled(
			klarerrs.Token(klarerrs.ErrPositiveSign, curr),
			"A leading '+' sign doesn't change a number",
		)
	case lexer.NotNot:
		count := p.countConsecutiveNot()
		err := klarerrs.Range(
			klarerrs.ErrDoubleNot,
			ranges.Offset(curr.Position, 0, uint32(count)),
		)
		err.SetParam("count", count)
		p.Error(err)
	}
	p.skipUntilBoundary()
}

// countConsecutiveNot counts the number of consecutive `!` tokens.
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

func (p *Parser) skipUntilBoundary() {
	var brackCount int
	for p.HasTokens() {
		switch p.CurrKind() {
		case lexer.Comma, lexer.Newline:
			if brackCount <= 0 {
				return
			}
		case lexer.LeftParenthesis, lexer.LeftBracket, lexer.LeftCurlyBrace, lexer.HashLeftCurlyBrace:
			brackCount++
		case lexer.RightParenthesis, lexer.RightBracket, lexer.RightCurlyBrace:
			brackCount--
			if brackCount < 0 {
				return
			}
		}
		p.Advance()
	}
}

// mismatchedLabelFormatError formats an error for a mismatch between type-only
// and labels-and-types parameters.
func mismatchedLabelFormatError(err *klarerrs.Error,
	prevIsTypeOnly bool, prevRange ranges.Range,
) {
	var msg string
	if prevIsTypeOnly {
		err.Label = "This parameter shouldn't have a label"
		msg = "This parameter already only has a type"
	} else {
		err.Label = "This parameter should have a label"
		msg = "This parameter already has a label"
	}
	err.Highlights = append(err.Highlights, klarerrs.Highlight{
		Range:   prevRange,
		Message: msg,
	})
}

// missingParamTypeAnnotError formats an error for a missing type annotation for
// a parameter or tuple item. missingParamTypeAnnotError sets err.Label.
func (p *Parser) missingParamTypeAnnotError(err *klarerrs.Error,
	kind string, labelCount int, lastParamRange ranges.Range,
) {
	err.SetParam("length", labelCount)
	if labelCount > 1 {
		err.Label = "These " + kind + "s need type annotations"
	} else {
		err.Label = "This " + kind + " needs a type annotation"
	}
	err.Highlights = append(err.Highlights, klarerrs.Highlight{
		Range:   lastParamRange,
		Message: "This " + kind + " already has a type",
	})
}
