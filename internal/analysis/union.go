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
