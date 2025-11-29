package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (*BaseNode) _node()              {}
func (b *BaseNode) Pos() ranges.Range { return b.Range }
func (b *BaseNode) SetPos(start, end lexer.Position) {
	b.Range = ranges.Range{Start: start, End: end}
}

// Values
func (*Boolean) value()     {}
func (*StringGroup) value() {}
func (*String) value()      {}
func (*Number) value()      {}
func (*List) value()        {}
func (*Object) value()      {}
func (*Class) value()       {}
func (*None) value()        {}
func (*Bad) value()         {}
func (*ArrowRef) value()    {}
