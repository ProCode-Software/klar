package ast

import "iter"

type AssignableIter = func(Assignable, *BadExpression) bool

var _ = [...]Destructurable{
	&Symbol{}, &Discard{}, &ListLiteral{}, &TupleLiteral{},
}

var _ = [...]Assignable{
	&Symbol{}, &Discard{}, &ListLiteral{}, &TupleLiteral{}, &SliceExpression{},
	&IndexExpression{}, &RestExpression{}, &MapLiteral{}, &BadExpression{},
}

// Destructurable
// =========

func (s *Symbol) Every(pred AssignableIter) bool {
	return pred(s, nil)
}

func (d *Discard) Every(pred AssignableIter) bool {
	return pred(&Symbol{d.BaseNode, "_"}, nil)
}

func (l *ListLiteral) Every(pred AssignableIter) bool {
	for _, item := range l.Items {
		if !validateAssignable(item, pred) {
			return false
		}
	}
	return true
}

func (l *TupleLiteral) Every(pred AssignableIter) bool {
	for _, item := range l.Values {
		if !validateAssignable(item, pred) {
			return false
		}
	}
	return true
}

func validateAssignable(node Expression, pred AssignableIter) bool {
	if dest, ok := node.(Assignable); ok {
		return dest.Every(pred)
	}
	return pred(nil, &BadExpression{
		BaseNode: BaseNode{Range: node.GetRange()},
		Value:    node,
	})
}

// TODO: Map as [Destructurable]
func (m *MapLiteral) Every(pred AssignableIter) bool {
	panic("map destructuring not implemented yet")
}

// [Assignable] but not [Destructurable]
// ==========

func (s *SliceExpression) Every(pred AssignableIter) bool { return pred(s, nil) }
func (s *IndexExpression) Every(pred AssignableIter) bool { return pred(s, nil) }
func (s *RestExpression) Every(pred AssignableIter) bool {
	return validateAssignable(s.Expression, pred)
}

// Error reported at parse-time
func (b *BadExpression) Every(pred AssignableIter) bool { return true }

func DestructureNames(node Assignable) iter.Seq2[*Symbol, *BadExpression] {
	return func(yield func(*Symbol, *BadExpression) bool) {
		if node == nil {
			return
		}
		node.Every(func(node Assignable, bad *BadExpression) bool {
			if bad != nil {
				return yield(nil, bad)
			}
			sym, ok := node.(*Symbol)
			if !ok {
				return yield(nil, &BadExpression{
					BaseNode: BaseNode{node.GetRange()},
					Value:    node,
				})
			}
			return yield(sym, bad)
		})
	}
}
