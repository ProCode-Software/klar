package parser

import "github.com/ProCode-Software/klar/internal/lexer"

// Lookaheads may be used to the parser to check what if after some tokens before
// conditionally parsing them.
const (
	failLookahead = iota
	continueLookahead
	breakLookahead
)

func (p *Parser) Lookahead(handler func(tok lexer.TokenType, last bool) int) bool {
	i := p.Index
loop:
	for brackCount := 0; ; i++ {
		tok := p.Tokens[i]
		switch tok.Kind {
		case lexer.RightBracket:
			brackCount--
			if brackCount == 0 {
				break loop
			}
		case lexer.LeftBracket:
			brackCount++
		case lexer.Stroke, lexer.Question:
			return true
		case lexer.Comma:
			if brackCount < 2 {
				return false
			}
		case lexer.LeftParenthesis, lexer.RightParenthesis,
			lexer.GreaterThan, lexer.LessThan, lexer.Identifier,
			lexer.Dot, lexer.Arrow, lexer.Ellipsis:
		default:
			return false
		}
	}
	return p.Tokens[i+1].Kind == lexer.LeftParenthesis
}
