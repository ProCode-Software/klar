package parser

import (
	"cmp"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	modeUnknown = iota
	typesOnly
	labelsAndTypes
)

func (p *Parser) ParseMethodParams() (params []*ast.MethodParam) {
	var (
		labels [][2]ast.Identifier
		mode   int // 0: unknown; 1: type only; 2: labels and types
	)
	setMode := func(m int) {
		if mode != m && mode != modeUnknown {
			err := errors.Token(errors.ErrMixTypeTupleLabels, p.Curr())
			prev := params[len(params)-2]
			mismatchedLabelFormat(err, len(prev.Names) == 0, prev.GetRange())
			p.Error(err)
		}
		mode = m
	}
	setParamRange := func() {
		mp := params[len(params)-1]
		if len(mp.Names) > 0 {
			mp.Range.Start = cmp.Or(mp.Names[0][0].Position, mp.Names[0][1].Position)
			mp.Range.End = mp.Type.GetRange().End
		} else {
			mp.Range = mp.Type.GetRange()
		}
	}

	for p.WhileNot(lexer.RightParenthesis) {
		if mode == typesOnly {
			params = append(params, &ast.MethodParam{
				Type: p.ParseType(DefaultTypeBindingPower),
			})
			setParamRange()
			if p.CurrKind() == lexer.Colon || isValidIdentOrDiscard(p.CurrKind()) {
				setMode(labelsAndTypes)
				p.Backup()
			}
			continue
		}
		if isValidIdentOrDiscard(p.CurrKind()) {
			var (
				name1   ast.Identifier
				name2   = p.ParseValidIdent()
				newMode = mode
			)
			if isValidIdentOrDiscard(p.CurrKind()) {
				newMode = labelsAndTypes
				name1, name2 = name2, p.ParseValidIdent()
			}
			labels = append(labels, [2]ast.Identifier{name1, name2})
			// Type annotation
			if p.CurrKind() == lexer.Colon {
				if name1.Name == "_" {
					p.Error(errors.Node(errors.ErrUnderscoreLabel, name1))
				}
				p.Advance()
				params = append(params, &ast.MethodParam{
					Names: labels,
					Type:  p.ParseType(DefaultTypeBindingPower),
				})
				setParamRange()
				newMode = labelsAndTypes
				labels = nil
			}
			setMode(newMode)
		} else {
			// A type
			params = append(params, &ast.MethodParam{
				Type: p.ParseType(DefaultTypeBindingPower),
			})
			setParamRange()
			setMode(typesOnly)
		}
		if p.CurrKind() != lexer.RightParenthesis {
			p.Expect(lexer.Comma)
		}
	}
	p.Expect(lexer.RightParenthesis)

	// Assign remaining labels as types
	if len(labels) > 0 {
		if mode == labelsAndTypes {
			first, last := labels[0], labels[len(labels)-1]
			err := errors.Range(errors.ErrMissingLabelsType, ranges.Range{
				Start: cmp.Or(first[0].Position, first[1].Position),
				End:   last[1].End(),
			})
			err.SetParam("length", len(labels))
			if len(labels) > 1 {
				err.Label = "These parameter need a type annotation"
			} else {
				err.Label = "This parameter needs a type annotation"
			}
			err.Highlights = append(err.Highlights, errors.Highlight{
				Range:   params[len(params)-1].Range,
				Message: "This parameter already has a type",
			})
			p.Error(err)
		}
		for _, lb := range labels {
			params = append(params, &ast.MethodParam{Type: lb[1].TypeAlias()})
			setParamRange()
		}
	}
	return params
}
