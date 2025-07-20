package ast

func (b *baseNode) Pos() int { return b.StartPos }
func (b *baseNode) End() int { return b.EndPos }

// Values
func (*Bool) value()      {}
func (*StringSeq) value() {}
func (*String) value()    {}
func (*Number) value()    {}
func (*Array) value()     {}
func (*Object) value()    {}
func (*Class) value()     {}
func (*Null) value()      {}
