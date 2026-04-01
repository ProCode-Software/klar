package lexer

import (
	"fmt"
	"io"
)

// Position represents a position in the source code. Line and Col are 1-based.
type Position struct{ Line, Col uint32 }

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

func (p Position) LitterDump(w io.Writer) {
	w.Write([]byte("{" + p.String() + "}"))
}

// IsZero reports whether p is the zero value.
func (p Position) IsZero() bool { return p.Line == 0 }

// Add returns a new Position with line and col added to p.
func (p Position) Add(line, col uint32) Position {
	return Position{Line: p.Line + line, Col: p.Col + col}
}

// Sub returns the new Position with line and col subtracted from p.
// The line and column are clamped to zero if they are negative.
func (p Position) Sub(line, col uint32) Position {
	return Position{Line: p.Line - min(p.Line, line), Col: p.Col - min(p.Col, col)}
}

// Add returns a new Position with n.Line and n.Col added to p.
func (p Position) AddPosition(n Position) Position {
	return Position{Line: p.Line + n.Line, Col: p.Col + n.Col}
}

// HasOffset reports whether p is equal to p1 + (line, col).
func (p Position) HasOffset(p1 Position, line, col uint32) bool {
	return p.Line == p1.Line+line && p.Col == p1.Col+col
}

// Offset returns a new Position with line and col added to p.
// The line and column are clamped to zero if they are negative.
// line and col may subtract from p's Line and Col.
func (p Position) Offset(line, col int) Position {
	return Position{uint32(int(p.Line) + line), uint32(int(p.Col) + col)}
}
