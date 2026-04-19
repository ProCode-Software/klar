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

func (p *Parser) parseMethodParams() (params []*ast.MethodParam) {
	var (
		labels []*ast.IdentifierPair
		mode   int // 0: unknown; 1: type only; 2: labels and types
	)
	setMode := func(m int) {
		// Error if mixing type-only and labels-and-types
		if mode != modeUnknown && mode != m {
			err := errors.Token(errors.ErrMixTypeTupleLabels, p.Curr())
			prev := params[len(params)-2]
			mismatchedLabelFormatError(err, len(prev.Names) == 0, prev.GetRange())
			p.Error(err)
		}
		mode = m
	}
	// Set ranges for appended parameters
	setParamRange := func() {
		if mp := params[len(params)-1]; len(mp.Names) > 0 {
			mp.Range.Start = cmp.Or(mp.Names[0].Label.Position, mp.Names[0].Name.Position)
			mp.Range.End = mp.Type.GetRange().End
		} else {
			mp.Range = mp.Type.GetRange()
		}
	}

	for p.WhileNot(lexer.RightParenthesis) {
		switch {
		case mode == typesOnly:
			params = append(params, &ast.MethodParam{
				Type: p.ParseType(DefaultTypeBindingPower),
			})
			setParamRange()
			if p.CurrKind() == lexer.Colon || isValidIdentOrDiscard(p.CurrKind()) {
				setMode(labelsAndTypes)
				p.Backup()
			}
		case isValidIdentOrDiscard(p.CurrKind()):
			// Possible label
			var (
				name1   ast.Identifier
				name2   = p.ParseValidIdent()
				newMode = mode
			)
			// Second label (parameter label)
			if isValidIdentOrDiscard(p.CurrKind()) {
				newMode = labelsAndTypes
				name1, name2 = name2, p.ParseValidIdent()
				if name1.Name == "_" {
					p.Error(errors.Node(errors.ErrUnderscoreLabel, name1))
				}
			}

			// Adds to a set of multiple parameters if there is no colon after
			labels = append(labels, &ast.IdentifierPair{Label: name1, Name: name2})

			// Type annotation
			if p.CurrKind() == lexer.Colon {
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
		default:
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

	// Assign remaining identifiers as types
	if len(labels) > 0 {
		// Labels missing a type
		if mode == labelsAndTypes {
			first, last := labels[0], labels[len(labels)-1]
			err := errors.Range(errors.ErrMissingLabelsType, ranges.Range{
				Start: cmp.Or(first.Label.Position, first.Name.Position),
				End:   last.Name.End(),
			})
			p.missingParamTypeAnnotError(err, "parameter", len(labels), params[len(params)-1].Range)
			p.Error(err)
		}
		for _, lb := range labels {
			params = append(params, &ast.MethodParam{Type: lb.Name.TypeAlias()})
			setParamRange()
		}
	}
	return params
}

func (p *Parser) ParseTupleType() *ast.TupleType {
	p.Expect(lexer.LeftParenthesis)
	var (
		t      = &ast.TupleType{}
		labels []ast.Identifier
		mode   int // 0: unknown; 1: type only; 2: labels and types
	)
	setMode := func(m int) {
		// Error if mixing type-only and labels-and-types
		if mode != modeUnknown && mode != m {
			prev, curr := t.Values[len(t.Values)-2], t.Values[len(t.Values)-1]
			err := errors.Range(errors.ErrMixTypeTupleLabels, curr.Range)
			mismatchedLabelFormatError(err, len(prev.Keys) == 0, prev.Range)
			p.Error(err)
		}
		mode = m
	}
	// Set ranges for appended parameters
	setParamRange := func() {
		if pa := t.Values[len(t.Values)-1]; len(pa.Keys) > 0 {
			pa.Range.Start = pa.Keys[0].Position
			pa.Range.End = pa.Value.GetRange().End
		} else {
			pa.Range = pa.Value.GetRange()
		}
	}

	for p.WhileNot(lexer.RightParenthesis) {
		switch {
		case mode == typesOnly:
			t.Values = append(t.Values, &ast.TypePair{
				Value: p.ParseType(DefaultTypeBindingPower),
			})
			setParamRange()
			if p.CurrKind() == lexer.Colon || isValidIdentOrDiscard(p.CurrKind()) {
				setMode(labelsAndTypes)
				p.Backup()
			}
		case isValidIdentOrDiscard(p.CurrKind()):
			// Adds to a set of multiple parameters if there is no colon after
			labels = append(labels, p.ParseValidIdent())

			// Type annotation
			if p.CurrKind() == lexer.Colon {
				p.Advance()
				t.Values = append(t.Values, &ast.TypePair{
					Keys:  labels,
					Value: p.ParseType(DefaultTypeBindingPower),
				})
				setParamRange()
				setMode(labelsAndTypes)
				labels = nil
			}
		default:
			// A type
			t.Values = append(t.Values, &ast.TypePair{Value: p.ParseType(DefaultTypeBindingPower)})
			setParamRange()
			setMode(typesOnly)
		}
		if p.CurrKind() != lexer.RightParenthesis {
			p.Expect(lexer.Comma)
		}
	}
	trailingComma := p.PeekBehind().Kind == lexer.Comma
	p.Expect(lexer.RightParenthesis)

	// Assign remaining labels as types
	if len(labels) > 0 {
		// Labels missing a type
		if mode == labelsAndTypes {
			err := errors.Range(errors.ErrMissingLabelsType, ranges.Range{
				Start: labels[0].Position,
				End:   labels[len(labels)-1].End(),
			})
			p.missingParamTypeAnnotError(err, "item", len(labels), t.Values[len(t.Values)-1].Range)
			p.Error(err)
		}
		for _, name := range labels {
			t.Values = append(t.Values, &ast.TypePair{Value: name.TypeAlias()})
			setParamRange()
		}
	}
	t.ParenOnly = len(t.Values) == 1 && len(t.Values[0].Keys) <= 1 && !trailingComma
	return t
}

// parseAssignableTypePairs parses a series of assignable expressions, optionally
// followed by an optional type and/or a default value. Types are optional for
// keys, but the format must be consistent.
func (p *Parser) parseAssignableTypePairs(pairs *[]*ast.AssignableTypePair,
	first ast.Assignable, isForLoop bool,
) {
	// parseSeries modifies the range for `first`, so use a manual for loop instead.
	var hasTypeAnnot bool
	for p.HasTokens() {
		pair := &ast.AssignableTypePair{}
		// Chained labels
		for p.HasTokens() {
			switch {
			case first != nil:
				pair.Keys = append(pair.Keys, first)
				first = nil
			case isForLoop:
				// If parsing variables for a 'for' loop, exclude 'in' expressions
				pair.Keys = append(pair.Keys, p.validateAssignable(
					p.ParseExpressionFilter(excludeIf(lexer.In), bpOf(lexer.In), 0),
				))
			default:
				pair.Keys = append(pair.Keys, p.ParseAssignable())
			}
			if p.CurrKind() != lexer.Comma {
				break
			}
			p.Advance()
		}
		// Type annotation
		if p.CurrKind() == lexer.Colon {
			p.Advance()
			pair.Type = p.ParseType(DefaultTypeBindingPower)
			hasTypeAnnot = true
		} else if hasTypeAnnot {
			err := errors.Range(errors.ErrMissingLabelsType, ranges.Range{
				Start: pair.Keys[0].GetRange().Start,
				End:   pair.Keys[len(pair.Keys)-1].GetRange().End,
			})
			p.missingParamTypeAnnotError(err, "variable", len(pair.Keys),
				(*pairs)[len(*pairs)-1].GetRange(),
			)
			p.Error(err)
		}
		// Default value: not allowed in for loop
		if p.isEqual(p.Curr()) && !isForLoop {
			if len(pair.Keys) > 1 {
				// Not allowed with multiple keys
				p.ErrorLabelled(
					errors.Range(errors.ErrChainedDefault, ranges.Range{
						pair.Keys[len(pair.Keys)-1].GetRange().Start,
						p.lastTokEnd(),
					}), "Unchain this parameter",
				)
			}
			p.Advance()
			pair.Value = p.ParseExpression(ExpressionBindingPower)
		}

		markStartEndPos(p, pair, pair.Keys[0].GetRange().Start)
		*pairs = append(*pairs, pair)
		if p.CurrKind() != lexer.Comma {
			break
		}
		p.Advance()
	}
}
