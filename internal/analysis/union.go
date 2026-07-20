package analysis

import (
	"strings"
)

type unionFlags uint16

const (
	// Union contains at least 1 optional type
	hasOptional unionFlags = 1 << iota
)

type Union struct {
	Types []Type
	Flags unionFlags
	fmset *FieldMethodSet
}

func NewUnion(types ...Type) Type {
	if len(types) == 1 {
		return types[0]
	}
	existing := make(map[Type]struct{})
	u := &Union{Types: make([]Type, 0, len(types))}
	insert := func(typ Type) {
		if _, ok := existing[typ]; !ok {
			u.Types = append(u.Types, typ)
			existing[typ] = struct{}{}
		}
	}
	for _, typ := range types {
		// Flatten unions
		if typ.Kind() == KindUnion {
			for _, typ := range As[*Union](typ).Types {
				insert(typ)
			}
		} else {
			insert(typ)
		}
	}
	if len(u.Types) == 1 {
		return u.Types[0]
	}
	return u
}

func (u *Union) Kind() Kind { return KindUnion }
func (u *Union) String() string {
	if len(u.Types) == 0 {
		return "empty union"
	}
	var b strings.Builder
	for _, t := range u.Types {
		b.WriteString(" | ")
		b.WriteString(t.String())
	}
	return b.String()[len(" | "):]
}
