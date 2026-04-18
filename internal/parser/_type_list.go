package parser

import (
	"iter"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

type listItem struct {
	label any
	typ   ast.Type
	value any
}

func (p *Parser) ParseTypeList(
	// If parseLabels returns nil, a type is immediately parsed instead of labels.
	parseLabels func() any,
	parseAfter func() any, // Can be used to parse an assignment
	terminator lexer.TokenType,
) iter.Seq[listItem] {
	after := func() any {
		if parseAfter != nil && p.isEqual(p.Curr()) {
			p.Advance()
			return parseAfter()
		}
		return nil
	}
	return func(yield func(listItem) bool) {
		var typesOnly bool
		for p.HasTokens() && p.CurrKind() != terminator {
			if typesOnly {
				// Types only
				if !yield(listItem{
					typ:   p.ParseType(DefaultBindingPower),
					value: after(),
				}) {
					return
				}
			} else {
				labels := parseLabels()
				if labels == nil {
					typesOnly = true
					continue
				}
				p.Expect(lexer.Colon)
				if !yield(listItem{
					label: labels,
					typ:   p.ParseType(DefaultBindingPower),
					value: after(),
				}) {
					return
				}
			}
			// Separator
			if p.CurrKind() != terminator {
				p.Expect(lexer.Comma)
			}
		}
		p.Expect(terminator)
	}
}
