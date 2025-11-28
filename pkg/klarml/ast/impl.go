package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (*baseNode) _node()              {}
func (b *baseNode) Pos() ranges.Range { return b.Range }
func (b *baseNode) SetPos(start, end lexer.Position) {
	b.Range = ranges.Range{Start: start, End: end}
}

// Values
func (*Bool) value()        {}
func (*StringGroup) value() {}
func (*String) value()      {}
func (*Number) value()      {}
func (*List) value()        {}
func (*Object) value()      {}
func (*Class) value()       {}
func (*None) value()        {}
