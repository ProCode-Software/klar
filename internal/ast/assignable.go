package ast

import "iter"

type DestructureIter = iter.Seq2[Identifier, *BadExpression]

func identOrError(e Expression) DestructureIter {
	if dest, ok := e.(Destructurable); ok {
		return dest.Names()
	}
	return func(yield func(Identifier, *BadExpression) bool) {
		yield(Identifier{}, &BadExpression{Value: e})
	}
}

func (s *Symbol) Names() DestructureIter {
	return func(yield func(Identifier, *BadExpression) bool) {
		yield(s.ToIdentifier(), nil)
	}
}

func (s *Discard) Names() DestructureIter {
	return func(yield func(Identifier, *BadExpression) bool) {
		yield(Identifier{Name: "_", Position: s.Range.Start, Len: 1}, nil)
	}
}

func (l *ListLiteral) Names() DestructureIter {
	return func(yield func(Identifier, *BadExpression) bool) {
		for _, item := range l.Items {
			for ident, err := range identOrError(item) {
				if !yield(ident, err) {
					return
				}
			}
		}
	}
}

func (l *TupleLiteral) Names() DestructureIter {
	return func(yield func(Identifier, *BadExpression) bool) {
		for _, item := range l.Values {
			for ident, err := range identOrError(item) {
				if !yield(ident, err) {
					return
				}
			}
		}
	}
}
