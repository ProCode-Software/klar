package ast

import (
	"unicode/utf8"

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
	_range   ranges.Range
}

func (i Identifier) Len() uint32 {
	return uint32(utf8.RuneCountInString(i.Name))
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
			i.Position.Line, i.Position.Col + i.Len(),
		}}
	}
	return i._range
}

func (i Identifier) Symbol() *Symbol {
	return &Symbol{Identifier: i.Name, BaseNode: BaseNode{Range: i._range}}
}

// Implementing [Node] just for error handling

// GetRange implements [Node]. It is an alias for i.Range
func (i Identifier) GetRange() ranges.Range {
	return i.Range()
}

// SetPos implements [Node]. It does not change i's range and only exists for
// error handling
func (i Identifier) SetPos(_, _ lexer.Position) {}
