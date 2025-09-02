package parser

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Lookaheads may be used to the parser to check what if after some tokens before
// conditionally parsing them.

const (
	failLookahead     = iota // The lookahead returns false
	continueLookahead        // The lookahead continues
	breakLookahead           // The lookahead evaluates the last token after
	passLookahead            // The lookahead returns true
)

// A lookahead function
type lookaheadHandler = func(tok lexer.TokenType, last bool, brackCount int) int

func (p *Parser) Lookahead(handler lookaheadHandler) bool {
	i, brackCount := p.Index, 0
loop:
	for ; ; i++ {
		tok := p.Tokens[i].Kind
		switch tok {
		case lexer.RightBracket, lexer.RightCurlyBrace, lexer.RightParenthesis:
			brackCount--
		case lexer.LeftCurlyBrace, lexer.LeftBracket, lexer.LeftParenthesis,
			lexer.HashLeftCurlyBrace:
			brackCount++
		}
		switch handler(tok, false, brackCount) {
		case failLookahead:
			return false
		case passLookahead:
			return true
		case continueLookahead:
			if tok == lexer.EOF {
				return false
			}
			continue loop
		case breakLookahead:
			if tok == lexer.EOF {
				return false
			}
			break loop
		}
	}
	return handler(p.Tokens[i+1].Kind, true, brackCount) == passLookahead
}

func passLookaheadIf(cond bool) int {
	if cond {
		return passLookahead
	}
	return failLookahead
}

func isListCast(tok lexer.TokenType, last bool, brackCount int) int {
	if last {
		return passLookaheadIf(tok == lexer.LeftParenthesis)
	}
	switch tok {
	case lexer.Stroke, lexer.Question:
		return passLookahead
	case lexer.Comma:
		if brackCount < 2 {
			return failLookahead
		}
	case lexer.LeftParenthesis, lexer.RightParenthesis,
		lexer.GreaterThan, lexer.LessThan, lexer.Identifier,
		lexer.Dot, lexer.Arrow, lexer.Ellipsis:
	default:
		if isValidIdentifier(tok) {
			break
		}
		return failLookahead
	}
	return continueLookahead
}

func isDestructureAssignment(tok lexer.TokenType, last bool, brackCount int) int {
	if last {
		switch tok {
		case lexer.Equal, lexer.ColonEqual,
			lexer.PlusEqual, lexer.MinusEqual, lexer.Colon, lexer.Comma, lexer.In:
			return passLookahead
		}
		return failLookahead
	}
	/* if brackCount == 0 {
		return breakLookahead
	} */
	switch tok {
	case lexer.Equal, lexer.ColonEqual,
		lexer.PlusEqual, lexer.MinusEqual, lexer.Colon, lexer.Comma, lexer.In:
		if brackCount < 1 {
			return passLookahead
		}
	case lexer.EndOfStatement:
		if brackCount < 1 {
			return failLookahead
		}
	case lexer.EOF:
		return failLookahead
	}
	return continueLookahead
}

func (p *Parser) isArrowFunction(tok lexer.TokenType, last bool, brackCount int) int {
	switch {
	case p.isWhenCase:
		return failLookahead // Arrow function not allowed in when cases
	case last:
		return passLookaheadIf(tok == lexer.Arrow)
	case brackCount == 0:
		return breakLookahead
	}
	switch tok {
	case lexer.RightParenthesis,
		lexer.RightCurlyBrace, lexer.RightBracket:
		if brackCount == 0 {
			return breakLookahead
		}
	case lexer.EOF:
		return failLookahead
	case lexer.EndOfStatement:
		if brackCount < 1 {
			return failLookahead
		}
	}
	return continueLookahead
}
