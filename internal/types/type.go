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
	GetFields() FieldMap
	GetMethods() MethodMap
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

type (
	Untyped     int
	UntypedEnum struct{ Name string }
)

const (
	UntypedInt Untyped = 11 + iota
	UntypedNil
	UntypedList // Empty list
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
	Generic   struct{ Name string }
	Interface struct{ Struct }
	Lambda    struct{ Function }
	Map       struct{ KeyType, ValueType Type }
	Result    struct{ SuccessType, FailureType Type }

	FieldMap  = map[string]Type
	MethodMap = map[string]Overloads
)

type Struct struct {
	Interface  bool
	Implements []*TypeDeclaration
	Order      []string
	Fields     map[string]Type
	Methods    map[string]Overloads
}
type Ref struct {
	Name  string
	Value *Type
}
type Function struct {
	Params []Param
	Return Type
}
type Overload struct {
	Position ranges.Range
	Function
}
type Overloads []Overload

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

func (o Overloads) Get(params []Param) (matching *Overload, found bool) {
outer:
	// Loop over each overload
	for _, overload := range o {
		if len(overload.Params) != len(params) {
			continue
		}
		// Compare each param
		for i, param := range params {
			if overload.Params[i] != param {
				continue outer
			}
		}
		return &overload, true
	}
	return nil, false
}

func (o *Overloads) Define(f Function, rang ranges.Range) (ok bool) {
	if _, found := o.Get(f.Params); found {
		return false
	}
	*o = append(*o, Overload{Function: f, Position: rang})
	return true
}

func (s *Struct) DefineMethod(name string, f Function, rang ranges.Range) (ok bool) {
	if overloads, found := s.Methods[name]; found {
		return overloads.Define(f, rang)
	}
	s.Methods[name] = Overloads{{Function: f, Position: rang}}
	return true
}
