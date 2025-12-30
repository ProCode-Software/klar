package argparse

import "fmt"

type FlagType int

const (
	_ FlagType = iota
	TypeBool
	TypeString
	TypeEnum
	TypeList
	TypeInt
	TypeFloat
)

type Flag struct {
	Type  FlagType // The type of the flag
	Value any      // The value of the flag
	Index int      // The index of the flag in the input arguments
	Set   bool     // Whether the flag was set, otherwise this flag is the default value
}

type Enum struct {
	key   string
	value any
}

// newDefaultFlag returns a [Flag] to be used as a default value
func newDefaultFlag(t FlagType, v any) *Flag {
	return &Flag{Type: t, Value: v, Index: -1}
}

func (f *Flag) Bool() bool {
	if f.Type != TypeBool {
		panic("flag is not of type bool")
	}
	return f.Value.(bool)
}

func (f *Flag) Int() int {
	if f.Type != TypeInt {
		panic("flag is not of type int")
	}
	return f.Value.(int)
}

func (f *Flag) Float() float64 {
	if f.Type != TypeFloat {
		panic("flag is not of type float")
	}
	return f.Value.(float64)
}

func (f *Flag) String() string {
	if f.Type != TypeString {
		return fmt.Sprintf("%v", f.Value)
	}
	return f.Value.(string)
}

func (f *Flag) Enum() *Enum {
	if f.Type != TypeEnum {
		panic("flag is not of type enum")
	}
	return f.Value.(*Enum)
}

func (e *Enum) Key() string { return e.key }
func (e *Enum) Value() any  { return e.value }
