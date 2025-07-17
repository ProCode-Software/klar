package ast

type Node interface {
	GetRange() Range
}
type Value interface {
	Node
	_value()
}

func (node *baseNode) GetRange() Range { return node.Range }

func (*Object) _value()         {}
func (*Array) _value()          {}
func (*Property) _value()       {}
func (*StringLiteral) _value()  {}
func (*ConcatString) _value()   {}
func (*NumericLiteral) _value() {}
func (*BoolLiteral) _value()    {}
func (*Namespace) _value()      {}
func (*VarRef) _value()         {}
func (*Bad) _value()            {}
