package analysis

import "fmt"

func (c *Checker) instantiateType(typ Type, params []Type) Type {
	// If typ is an alias, preserve its name
	res := typ
	switch typ := Underlying(typ).(type) {
	case *Enum:

	case *Overload:
	// TODO: Add Result and Task?
	default:
		panic(fmt.Sprintf("can't instantiate %T", typ))
	}
	return res
}
