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

func (CoreType) Type() {}
func (Union) Type()    {}
func (Struct) Type()   {}
func (Alias) Type()    {}
