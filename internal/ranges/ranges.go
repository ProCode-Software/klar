// Package ranges provides utilities for working with source code positions and ranges.
// It defines types and functions to represent, manipulate, and query ranges of text
// based on line and column positions, typically used in lexers and parsers.
package ranges

import (
	"cmp"
	"fmt"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// Position is an alias for [lexer.Position]
type Position = lexer.Position

// A Range represents a span of text, defined by a start and end [Position].
type Range struct {
	Start, End Position
}

func NewRange(sl, sc, el, ec uint32) Range {
	return Range{Start: Position{sl, sc}, End: Position{el, ec}}
}

// FromToken returns a new Range that is the position and length of token t.
// If the token is multiline, this will only work with an 'end' attribute.
func FromToken(t lexer.Token) Range {
	return Range{Start: t.Position, End: t.End()}
}

// Positions
// ===========

// Min returns the lowest position out of p1 and p2. If they are equal, Min returns p1.
func Min(p1, p2 Position) Position {
	if p1.Line < p2.Line {
		return p1
	} else if p1.Line == p2.Line && p1.Col <= p2.Col {
		return p1
	}
	return p2
}

// Offset returns a [Range] that starts at start and has a length of endLine and endCol.
func Offset(start Position, endLine, endCol uint32) Range {
	return Range{
		Start: start,
		End:   Position{Line: start.Line + endLine, Col: start.Col + endCol},
	}
}

// Ranges
// =============

// Between returns a [Range] that starts at r1.Start and ends at r2.End.
func Between(r1, r2 Range) Range {
	return Range{Start: r1.Start, End: r2.End}
}

// In reports whether p is in the range r.
func (r Range) PosIn(p Position) bool {
	if p.Line < r.Start.Line || p.Line > r.End.Line {
		return false
	}
	return p.Col >= r.Start.Col && p.Col <= r.End.Col
}

// RangeIn reports whether r2 is entirely inside r.
func (r Range) RangeIn(r2 Range) bool {
	return r.PosIn(r2.Start) && r.PosIn(r2.End)
}

// LineIn reports whether line l is in the range r.
func (r Range) LineIn(l uint32) bool {
	return r.Start.Line <= l && l <= r.End.Line
}

func (r Range) IsSingleLine() bool {
	return r.Start.Line == r.End.Line
}

// LineLength returns the column length between r.Start and r.End. If r is
// multiline, LineLength returns 0.
func (r Range) LineLength() uint32 {
	if !r.IsSingleLine() {
		return 0
	}
	return r.End.Col - r.Start.Col
}

func (r Range) String() string {
	return fmt.Sprintf("%s-%s", r.Start, r.End)
}

// IsZero reports whether r is the zero value.
func (r Range) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero()
}

// Lines returns the number of lines r covers. If r is on a single line, Lines returns 1.
func (r Range) Lines() uint32 {
	return r.End.Col - r.Start.Col + 1
}

// TokenIntersects reports whether t intersects r at any point.
func (r Range) TokenIntersects(t lexer.Token) bool {
	return r.PosIn(t.Position) || r.PosIn(t.End())
}

func Compare(a, b Range) int {
	return cmp.Or(
		cmp.Compare(a.Start.Line, b.Start.Line),
		cmp.Compare(a.Start.Col, b.Start.Col),
		cmp.Compare(a.End.Line, b.End.Line),
		cmp.Compare(a.End.Col, b.End.Col),
	)
}

// A FileRange represents a range of positions in a file.
type FileRange struct {
	Range
	File string
}

func (r FileRange) String() string {
	return fmt.Sprintf("%s-%s", r.File, r.Range)
}

func (r FileRange) FilePos() FilePos {
	return FilePos{Position: r.Start, File: r.File}
}

// File Positions
// ==================

// A FilePos represents a position in a file.
type FilePos struct {
	Position
	File string
}

func (r FilePos) String() string {
	return fmt.Sprintf("%s:%s", r.File, r.Position)
}

// Rel returns a formatted position with the file path dropped if p.File == to.
func (p FilePos) Rel(to string) string {
	if p.File == to {
		return p.Position.String()
	}
	return p.String()
}
