// Package ranges provides utilities for working with source code positions and ranges.
// It defines types and functions to represent, manipulate, and query ranges of text
// based on line and column positions, typically used in lexers and parsers.
package ranges

import (
	"fmt"
	"slices"

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

func IsZeroPosition(p Position) bool {
	return p.Line == 0
}

func getTokEnd(t lexer.Token) (end Position, ok bool) {
	if t.Attributes != nil {
		if end, ok := t.Attributes["end"].(Position); ok && !IsZeroPosition(end) {
			return end, true
		}
	}
	return
}

// FromToken returns a new Range that is the position and length of token t.
// If the token is multiline, this will only work with an 'end' attribute.
func FromToken(t lexer.Token) Range {
	if t.Attributes != nil {
		if end, ok := getTokEnd(t); ok {
			return Range{t.Position, end}
		}
	}
	return Range{Start: t.Position, End: Position{
		Line: t.Position.Line,
		Col:  t.Position.Col + t.Len(),
	}}
}

func TokenEnd(t lexer.Token) Position {
	if end, ok := getTokEnd(t); ok {
		return end
	}
	return Position{Line: t.Position.Line, Col: t.Position.Col + t.Len()}
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
func Sub(p Position, line, col uint32) Position {
	return Position{Line: p.Line - line, Col: p.Col - col}
}

func Offset(start Position, endLine, endCol uint32) Range {
	return Range{
		Start: start,
		End:   Position{Line: start.Line + endLine, Col: start.Col + endCol},
	}
}

// Add returns a new Position with line and col added to p.
func Add(p Position, line, col uint32) Position {
	return Position{Line: p.Line + line, Col: p.Col + col}
}

// Add returns a new Position with n.Line and n.Col added to p.
func AddPosition(p Position, n Position) Position {
	return Add(p, n.Line, n.Col)
}

// HasOffset reports whether pos is equal to from plus line and col.
func HasOffset(pos Position, from Position, line, col uint32) bool {
	return pos.Line == from.Line+line && pos.Col == from.Col+col
}

// Ranges
// =============

// Between returns a [Range] that starts at r1.Start and ends at r2.End.
func Between(r1, r2 Range) Range {
	return Range{Start: r1.Start, End: r2.End}
}

// FromPosition returns a [Range] that starts at start and ends at end.
func FromPosition(start, end Position) Range {
	return Range{Start: start, End: end}
}

// In reports whether p is in the range r.
func (r Range) PosIn(p Position) bool {
	if p.Line < r.Start.Line || p.Line > r.End.Line {
		return false
	}
	return p.Col >= r.Start.Col &&
		p.Col <= r.End.Col
}

// RangeIn reports whether r2 is entirely inside r.
func (r Range) RangeIn(r2 Range) bool {
	return r.PosIn(r2.Start) && r.PosIn(r2.End)
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
	return IsZeroPosition(r.Start) && IsZeroPosition(r.End)
}

// Lines returns the number of lines r covers. If r is on a single line, Lines returns 1.
func (r Range) Lines() uint32 {
	return r.End.Col - r.Start.Col + 1
}

func Sort(ranges ...Range) []Range {
	slices.SortFunc(ranges, func(a, b Range) int {
		if a.Start.Line < b.Start.Line {
			return -1
		} else if a.Start.Line > b.Start.Line {
			return 1
		}
		if a.Start.Col < b.Start.Col {
			return -1
		} else if a.Start.Col > b.Start.Col {
			return 1
		}
		return 0
	})
	return ranges
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
