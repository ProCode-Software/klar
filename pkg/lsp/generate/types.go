package main

import "github.com/ProCode-Software/klar/pkg/lsp/rpc"

type MessageDirection string

const (
	ClientToServer MessageDirection = "clientToServer"
	ServerToClient MessageDirection = "serverToClient"
	Both           MessageDirection = "both"
)

type Type struct {
	Kind TypeKind `json:"kind"`

	// Intersection of all possible type kinds

	Name    string `json:"name,omitempty"`    // [BaseTypes] if 'base', string if 'reference'
	Element *Type  `json:"element,omitempty"` // Array
	// Map. Represents URI | DocumentUri | string | integer | ReferenceType
	Key   *Type  `json:"key,omitempty"`
	Items []Type `json:"items,omitempty"` // and | or | tuple
	// 	- [Type] if map
	// 	- [bool] if booleanLiteral
	// 	- [string] if stringLiteral
	// 	- [int] if integerLiteral
	// 	- [StructureLiteral] if literal
	// Put StructureLiteral first so it is tried before a generic map
	Value *rpc.Union2[*StructureLiteral, rpc.Union2[Type, any]] `json:"value,omitempty"`
}

type TypeKind string

const (
	KindBase      TypeKind = "base"
	KindReference TypeKind = "reference"
	KindArray     TypeKind = "array"
	KindMap       TypeKind = "map"
	KindAnd       TypeKind = "and" // No occurrences
	KindOr        TypeKind = "or"
	// As of LSP 3.18, 1 occurrence (item types are commom)
	KindTuple TypeKind = "tuple"
	// As of LSP 3.18, 2 occurrences. Both empty objects
	KindStructLiteral TypeKind = "literal"
	KindStringLiteral TypeKind = "stringLiteral"
	// No occurrences as of LSP 3.18
	KindIntegerLiteral TypeKind = "integerLiteral"
	KindBooleanLiteral TypeKind = "booleanLiteral"
)

type BaseType struct {
	Kind string    `json:"kind"` // "base"
	Name BaseTypes `json:"name"`
}

type BaseTypes string

const (
	URI         BaseTypes = "URI"
	DocumentUri BaseTypes = "DocumentUri"
	Integer     BaseTypes = "integer"
	UInteger    BaseTypes = "uinteger"
	Decimal     BaseTypes = "decimal"
	RegExp      BaseTypes = "RegExp"
	String      BaseTypes = "string"
	Boolean     BaseTypes = "boolean"
	Null        BaseTypes = "null"
)

type StructureLiteral struct {
	BaseDecl
	Name       struct{}   `json:"-"`
	Properties []Property `json:"properties"`
}

func (t Type) Equal(t2 Type) bool {
	return t.Kind == t2.Kind && t.Name == t2.Name
}
