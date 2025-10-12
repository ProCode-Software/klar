package ast

func (b *baseNode) Range() (int, int)       { return b.Start, b.End }
func (b *baseNode) SetRange(start, end int) { b.Start, b.End = start, end }

// Values
func (*Bool) value()      {}
func (*StringSeq) value() {}
func (*String) value()    {}
func (*Number) value()    {}
func (*Array) value()     {}
func (*Object) value()    {}
func (*Class) value()     {}
func (*Nil) value()       {}
