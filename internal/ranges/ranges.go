// Package ranges provides utilities for working with source code positions and ranges.
// It defines types and functions to represent, manipulate, and query ranges of text
// based on line and column positions, typically used in lexers and parsers.
package ranges

import (
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Position is an alias for [lexer.Position]
type Position = lexer.Position

// A Range represents a span of text, defined by a start and end [Position].
type Range struct {
	Start, End Position
}

// FromToken returns a new Range that is the position and length of token t.
func FromToken(t lexer.Token) Range {
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

// Add returns a new Position with line and col added to p.
func AddPosition(p Position, n Position) Position {
	return Add(p, n.Line, n.Col)
}

// In reports whether p is in the range r.
func (r Range) In(p Position) bool {
	if p.Line < r.Start.Line || p.Line > r.End.Line {
		return false
	}
	return p.Col >= r.Start.Col &&
		p.Col <= r.End.Col
}
