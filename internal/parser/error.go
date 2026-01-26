package parser

import "github.com/ProCode-Software/klar/internal/lexer"

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
