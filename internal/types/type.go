package types

type Type interface {
	Type()
}

type CoreType int

const (
	_ CoreType = iota
	String
	Int
	Float
	Bool
	List
	Function
	Map
	InvalidType
)

type (
	Union    struct{ Options []Type }
	Struct   struct{ Fields map[string]Type }
	Alias    struct{ For Type }
	Optional struct{ Underlying Type }
	Enum     struct {
		ValueType Type
		Members   map[string]any
	}
	Value struct {
		Type  Type
		Value any
	}
)

func (CoreType) Type() {}
func (Union) Type()    {}
func (Struct) Type()   {}
func (Alias) Type()    {}
func (Optional) Type() {}
func (Enum) Type()     {}
