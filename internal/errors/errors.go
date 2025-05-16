package errors

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

const (
	_ = iota
	UnexpectedTokenError
	ExpectedTokenError
	UndefinedReferenceError
)

type KlarError struct {
	Position lexer.Position
	Type     int
	Token    ast.ASTItem
}
