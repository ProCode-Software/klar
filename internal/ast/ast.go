package ast

import (
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/ranges"
)

//go:generate stringer -type=PrimitiveTypeName -linecomment
//go:generate go run ../cmd/asttempl

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

type Type interface {
	Node
	_type()
}

// ReservedIdent is the set of keywords that cannot be used as variable names.
var ReservedIdent = []lexer.TokenType{
	lexer.And, lexer.Await, lexer.Boolean, lexer.Stop, lexer.For, lexer.Func,
	lexer.Go, lexer.In, lexer.Next, lexer.Nil, lexer.Or,
	lexer.Return, lexer.Type, lexer.When, lexer.While,
}

// Keywords that are used before declarations.
var Modifiers = []lexer.TokenType{lexer.Public}

// Primitives
type PrimitiveTypeName int

const (
	PrimitiveAny     PrimitiveTypeName = iota // Any
	PrimitiveString                           // String
	PrimitiveInt                              // Int
	PrimitiveFloat                            // Float
	PrimitiveBool                             // Bool
	PrimitiveNothing                          // Nothing
	PrimitiveResult                           // Result
	PrimitiveError                            // Error
)

var PrimitiveTypeMap = map[string]PrimitiveTypeName{
	"Any":     PrimitiveAny,
	"String":  PrimitiveString,
	"Int":     PrimitiveInt,
	"Float":   PrimitiveFloat,
	"Bool":    PrimitiveBool,
	"Nothing": PrimitiveNothing,
	"Result":  PrimitiveResult,
	"Error":   PrimitiveError,
}

func (p *Program) Deps(yield func(imports.ImportPath) bool) {
	for _, stmt := range p.Body {
		imp, ok := stmt.(*ImportStatement)
		if !ok || !yield(imp.Module) {
			return
		}
	}
}
