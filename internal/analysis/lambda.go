package analysis

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
)

type Lambda struct {
	Params   []Type
	Variadic bool
	Return   Type
}

func (*Lambda) Kind() Kind { return KindFunction }
func (l *Lambda) String() string {
	var b strings.Builder
	b.WriteString("func(")
	for i, param := range l.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		if i == len(l.Params)-1 && l.Variadic {
			b.WriteString("...")
		}
		b.WriteString(param.String())
	}
	b.WriteByte(')')
	return b.String()
}
func (l *Lambda) Underlying() Type { return l }

func (c *Checker) checkFunctionType(expr *ast.FunctionType, ctx *Context) Type {
	return InvalidType
}
