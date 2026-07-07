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

func (o Operator) End() lexer.Position {
	pos := o.Position
	pos.Col += o.Len()
	return pos
}

// String returns the string representation of o.
func (o Operator) String() string { return o.Kind.String() }

// Equal returns true if o equals b.
func (a Operator) Equal(b Operator) bool { return a.Kind == b.Kind }

// Len returns the length of o.
func (o Operator) Len() uint32 { return uint32(len(o.Kind.String())) }

func (o Operator) Range() ranges.Range { return ranges.Range{o.Position, o.End()} }

// If o is a compound assignment operator (e.g. '+=', '*='), returns the uncompound
// operator ('+' and '*' respectively).
func (o Operator) Uncompound() Operator {
	uncompounded, ok := map[lexer.TokenType]lexer.TokenType{
		lexer.PlusEqual:     lexer.Plus,
		lexer.MinusEqual:    lexer.Minus,
		lexer.AsteriskEqual: lexer.Asterisk,
		lexer.SlashEqual:    lexer.SlashEqual,
		lexer.PercentEqual:  lexer.Percent,
		lexer.CaretEqual:    lexer.Caret,
	}[o.Kind]
	if ok {
		o.Kind = uncompounded
	}
	return o
}

// An Identifier represents a name in the source code.
type Identifier struct {
	Name     string
	Position lexer.Position
	Len      uint32
	_range   ranges.Range
}

// IsZero returns true if i is the zero value.
func (i Identifier) IsZero() bool { return i.Name == "" && i.Position.IsZero() }

// IsDiscard returns true if i is a discard identifier.
func (i Identifier) IsDiscard() bool { return i.Name == "_" }

// BaseNode returns i.Range() as a [BaseNode]
func (i Identifier) BaseNode() BaseNode { return BaseNode{Range: i.Range()} }

// End returns the end position of i.
func (i Identifier) End() lexer.Position { return i.Range().End }

// String returns i.Name.
func (i Identifier) String() string { return i.Name }

// Range returns the range of i.
func (i Identifier) Range() ranges.Range {
	if i._range.Start.Line == 0 {
		i._range = ranges.Range{i.Position, lexer.Position{
			i.Position.Line, i.Position.Col + i.Len,
		}}
	}
	return i._range
}

// Symbol returns i as a [Symbol].
func (i Identifier) Symbol() *Symbol {
	return &Symbol{Identifier: i.Name, BaseNode: i.BaseNode()}
}

// TypeAlias returns i as a [TypeAlias].
func (i Identifier) TypeAlias() *TypeAlias { return (*TypeAlias)(i.Symbol()) }

// Implementing [Node] just for error reporting
// =========

// GetRange implements [Node]. GetRange is i.Range
func (i Identifier) GetRange() ranges.Range { return i.Range() }

// SetPos implements [Node]. It does not change i's range.
func (i Identifier) SetPos(start, end lexer.Position) {}

// Walk implements [Node].
func (i Identifier) Walk(v Visitor, c *Cursor) StopCode { return walkFields(v, i, c) }

func (a Identifier) Equal(b2 Node) bool {
	b, ok := b2.(Identifier)
	return ok && a.Name == b.Name
}

func equalSlice[T any](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	if a, ok := any(a).([]Node); ok {
		b := any(b).([]Node)
		for i := range a {
			if !a[i].Equal(b[i]) {
				return false
			}
		}
		return true
	}
	for i := range a {
		if any(a[i]) != any(b[i]) {
			return false
		}
	}
	return true
}

func (s *Symbol) ToIdentifier() Identifier {
	return Identifier{
		Name:     s.Identifier,
		Position: s.Range.Start,
		Len:      s.Range.LineLength(),
	}
}
