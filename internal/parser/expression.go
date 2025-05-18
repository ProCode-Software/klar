package parser

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// For debugging purposes
func noHandlerError(p *Parser, nudOrLED string) {
	panic(fmt.Sprintf("Unexpected token '%s' (expected %s handler for %s)\n", p.CurrentToken().Source, nudOrLED, lexer.TokenTypes[p.CurrentTokenKind()]))
}

func (p *Parser) ParseExpression(bp BindingPower) ast.ASTItem {
	kind := p.CurrentTokenKind()
	left, handled := p.handleNUD(kind)
	if !handled {
		noHandlerError(p, "NUD")
	}
	for BindingPowerMap[p.CurrentTokenKind()] > bp {
		kind = p.CurrentTokenKind()
		left, handled = p.handleLED(kind, left, bp)
		if !handled {
			noHandlerError(p, "LED")
		}
	}
	return left
}