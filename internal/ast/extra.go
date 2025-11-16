package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Operator
type Operator struct {
	Kind     lexer.TokenType
	Position lexer.Position
}

func (o Operator) Len() uint32 {
	return uint32(len(o.Kind.String()))
}

func (o Operator) End() lexer.Position {
	pos := o.Position
	pos.Col += o.Len()
	return pos
}

// Identifier
type Identifier struct {
	Name     string
	Position lexer.Position
	Len      uint32
	_range   ranges.Range
}

func (i *Identifier) IsZero() bool {
	return i.Name == "" && i.Position.Line == 0
}

func (i Identifier) End() lexer.Position {
	return i._range.End
}

// BaseNode returns i.Range() as a [BaseNode]
func (i Identifier) BaseNode() BaseNode {
	return BaseNode{Range: i.Range()}
}

func (i Identifier) Range() ranges.Range {
	if i._range.Start.Line == 0 {
		i._range = ranges.Range{i.Position, lexer.Position{
			i.Position.Line, i.Position.Col + i.Len,
		}}
	}
	return i._range
}

func (i Identifier) Symbol() *Symbol {
	return &Symbol{Identifier: i.Name, BaseNode: BaseNode{Range: i.Range()}}
}

// Implementing [Node] just for error reporting

// GetRange implements [Node]. GetRange is i.Range
func (i Identifier) GetRange() ranges.Range {
	return i.Range()
}

// SetPos implements [Node]. It does not change i's range.
func (i Identifier) SetPos(start, end lexer.Position) {}
