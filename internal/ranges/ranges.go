// Package ranges provides utilities for working with source code positions and ranges.
// It defines types and functions to represent, manipulate, and query ranges of text
// based on line and column positions, typically used in lexers and parsers.
package ranges

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// Position is an alias for [lexer.Position]
type Position = lexer.Position

// A Range represents a span of text, defined by a start and end [Position].
type Range struct {
	Start, End Position
}

func NewRange(sl, sc, el, ec int) Range {
	return Range{Start: Position{sl, sc}, End: Position{el, ec}}
}

func IsZeroPosition(p Position) bool {
	return p.Line == 0
}

// FromToken returns a new Range that is the position and length of token t.
// If the token is multiline, this will only work with an 'end' attribute.
func FromToken(t lexer.Token) Range {
	if t.Attributes != nil {
		if end, ok := t.Attributes["end"].(Position); ok && !IsZeroPosition(end) {
			return Range{t.Position, end}
		}
	}
	return Range{Start: t.Position, End: Position{
		Line: t.Position.Line,
		Col:  t.Position.Col + len(t.Source),
	}}
}

// Min returns the lowest position out of p1 and p2. If they are equal, Min returns p1.
func Min(p1, p2 Position) Position {
	if p1.Line < p2.Line {
		return p1
	} else if p1.Line == p2.Line && p1.Col <= p2.Col {
		return p1
	}
	return p2
}

// Sub returns the new Position with line and col subtracted from p.
// The line and column are clamped to zero if they are negative.
func Sub(p Position, line, col int) Position {
	newPos := Position{Line: p.Line - line, Col: p.Col - col}
	if newPos.Col < 0 {
		newPos.Col = 0
	}
	if newPos.Line < 0 {
		newPos.Line = 0
	}
	return newPos
}

// Add returns a new Position with line and col added to p.
func Add(p Position, line, col int) Position {
	return Position{Line: p.Line + line, Col: p.Col + col}
}

func AddPosition(p Position, n Position) Position {
	return Add(p, n.Line, n.Col)
}

func Between(r1, r2 Range) Range {
	return Range{Start: r1.Start, End: r2.End}
}

func BetweenPos(r1, r2 Position) Range {
	return Range{Start: r1, End: r2}
}

// In reports whether p is in the range r.
func (r Range) PosIn(p Position) bool {
	if p.Line < r.Start.Line || p.Line > r.End.Line {
		return false
	}
	return p.Col >= r.Start.Col &&
		p.Col <= r.End.Col
}

func (r Range) RangeIn(r2 Range) bool {
	return r.PosIn(r2.Start) && r.PosIn(r2.End)
}

func (r Range) IsSingleLine() bool {
	return r.Start.Line == r.End.Line
}

func (r Range) LineLength() int {
	if !r.IsSingleLine() {
		return -1
	}
	return r.End.Col - r.Start.Col
}

func (r Range) String() string {
	return fmt.Sprintf("%s:%s", r.Start, r.End)
}

func (r Range) IsZero() bool {
	return IsZeroPosition(r.Start) && IsZeroPosition(r.End)
}
