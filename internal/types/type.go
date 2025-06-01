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
	ErrorType
)

type Union struct {
	Options []Type
}
type Struct struct {
	Fields map[string]Type
}
type Alias struct {
	For Type
}
type Optional struct {
	Underlying Type
}
type Enum struct {
	Members []string
}

func (CoreType) Type() {}
func (Union) Type()    {}
func (Struct) Type()   {}
func (Alias) Type()    {}
func (Optional) Type() {}
func (Enum) Type()     {}
