package ranges

import "github.com/ProCode-Software/klar/internal/lexer"

type Position = lexer.Position

func Min(p1, p2 Position) Position {
	if p1.Line < p2.Line {
		return p1
	} else if p1.Line == p2.Line && p1.Col <= p2.Col {
		return p1
	}
	return p2
}

func Sub(pos Position, line, col int) Position {
	newPos := Position{Line: pos.Line - line, Col: pos.Col - col}
	if newPos.Col < 0 {
		newPos.Col = 0
	}
	if newPos.Line < 0 {
		newPos.Line = 0
	}
	return newPos
}

