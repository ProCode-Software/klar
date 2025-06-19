package types

import (
	"fmt"
	"strings"
)

func stringGeneric(typ string, k, v Type) string {
	return fmt.Sprintf("%s<%s, %s>", typ, k, v)
}

func (g Generic) String() string {
	return g.Name
}

func (l List) String() string {
	return fmt.Sprintf("[%s]", l.Of)
}

func (o Optional) String() string {
	return fmt.Sprintf("%s?", o.Underlying)
}

func (u Union) String() string {
	var b strings.Builder
	for i, item := range u.Options {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(fmt.Sprintf("%s", item))
	}
	return b.String()
}

func (t Tuple) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, item := range t.Items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%s", item))
	}
	b.WriteByte(')')
	return b.String()
}

func (l Lambda) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, param := range l.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		if param.Variadic {
			b.WriteString("...")
		}
		b.WriteString(fmt.Sprintf("%s", param.Type))
	}
	b.WriteString(") -> ")
	b.WriteString(fmt.Sprintf("%s", l.Return))
	return b.String()
}

func (m Map) String() string {
	return stringGeneric("Map", m.KeyType, m.ValueType)
}

func (r Result) String() string {
	return stringGeneric("Map", r.SuccessType, r.FailureType)
}

func (r Ref) String() string {
	return r.Name
}

func (f Function) String() string {
	var b strings.Builder
	b.WriteByte('(')
	for i, param := range f.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		if param.Label != "" {
			b.WriteString(param.Label + ": ")
		}
		if param.Variadic {
			b.WriteString("...")
		}
		b.WriteString(fmt.Sprintf("%s", param.Type))
	}
	b.WriteByte(')')
	if f.Return != Nothing && f.Return != nil {
		b.WriteString(" -> ")
		b.WriteString(fmt.Sprintf("%s", f.Return))
	}
	return b.String()
}

func (f Function) StringNamed(name string) string {
	return name + f.String()
}