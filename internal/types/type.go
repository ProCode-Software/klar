package types

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type TypeDeclaration struct {
	Type       Type
	Used       bool
	Position   ranges.Range
	Attributes map[string]any
}

func (TypeDeclaration) Exportable_() {}

type Type interface {
	type_()
}
type HasFields interface {
	Type
	GetFields() map[string]Type
	GetMethods() map[string]Function
}

//go:generate stringer -type=CoreType -linecomment
type CoreType int

const (
	_ CoreType = iota
	Any
	String
	Int
	Float
	Bool
	Nothing
	Error

	InvalidType // Invalid
	Self
)

var PrimitiveMap = map[ast.PrimitiveTypeName]Type{
	ast.PrimitiveBool:    Bool,
	ast.PrimitiveAny:     Any,
	ast.PrimitiveInt:     Int,
	ast.PrimitiveFloat:   Float,
	ast.PrimitiveString:  String,
	ast.PrimitiveNothing: Nothing,
	ast.PrimitiveError:   Error,
	ast.PrimitiveMap:     Map{Any, Any},
	ast.PrimitiveResult:  Result{Nothing, Error},
}

type (
	List      struct{ Of Type }
	Tuple     struct{ Items []Type }
	Union     struct{ Options []Type }
	Optional  struct{ Underlying Type }
	Interface struct{ Struct }
	Lambda    struct{ Function }
	Map       struct{ KeyType, ValueType Type }
	Result    struct{ SuccessType, FailureType Type }
)

type Struct struct {
	Interface  bool
	Implements []*TypeDeclaration
	Order      []string
	Fields     map[string]Type
	Methods    map[string]Function
}
type Ref struct {
	Name  string
	Value Type
}
type Function struct {
	Params []Param
	Return Type
}
type Param struct {
	Label    string
	Type     Type
	Variadic bool
}
type Enum struct {
	ValueType Type
	Members   map[string]any
}
type Value struct {
	Type  Type
	Value any
}
