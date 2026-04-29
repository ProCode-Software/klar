package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// All AST tokens implement the Node interface.
type Node interface {
	GetRange() ranges.Range
	SetPos(start, end lexer.Position)
	// Equality checks disregard [BaseNode]s and positions.
	Equal(b Node) bool
	Walk(v Visitor, parent *Cursor) StopCode
}

// All EOS-terminated statement AST tokens implement the Statement interface.
type Statement interface {
	Node
	stmt()
}

// All expression AST tokens implement the Expression interface.
type Expression interface {
	Node
	expr()
}
