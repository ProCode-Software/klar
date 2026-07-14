package analysis

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Lambda struct {
	Params []Type // Variadic param isn't a List
	// Arity can be calculated lazily, but if the declaration contains
	// params with default values, this should be provided manually.
	arity    Arity
	Variadic bool
	Return   Type
	Complete bool
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
	if l.Return != nil && l.Return.Kind() != NothingType {
		b.WriteString(" -> ")
		b.WriteString(l.Return.String())
	}
	return b.String()
}
func (l *Lambda) Underlying() Type { return l }
func (l *Lambda) Arity() Arity {
	if l.arity != (Arity{0, 0}) {
		return l.arity
	}
	a := Arity{}
	for _, par := range l.Params {
		if par.Kind() != KindOptional {
			a.MinParams++
		}
		a.MaxParams++
	}
	if l.Variadic {
		a.MaxParams = -1
	}
	l.arity = a
	return a
}

func (c *Checker) checkFunctionType(expr *ast.FunctionType, ctx *Context) Type {
	l := &Lambda{Params: make([]Type, 0, len(expr.Parameters.Values))}
	for i, pair := range expr.Parameters.Values {
		typ, variadic := c.parseTypeOrVariadic(pair.Value, ctx)
		for range max(len(pair.Keys), 1) {
			l.Params = append(l.Params, typ)
		}
		if variadic {
			l.Variadic = true
			// Ensure this is the last and only paramerer
			if len(pair.Keys) > 1 || i < len(expr.Parameters.Values)-1 {
				var node ast.Node
				var after ranges.Range
				if len(pair.Keys) > 1 {
					// `(k1, k2: ...Int, _: Int)`
					node = pair.Keys[0]
					after = ranges.Range{
						pair.Keys[1].Position,
						expr.Parameters.Values[len(expr.Parameters.Values)-1].Range.End,
					}
				} else {
					// `(k: ...Int, _: Int)`
					node = pair
					after = ranges.FromSlice(expr.Parameters.Values[i+1:])
				}
				err := klarerrs.Node(klarerrs.ErrVariadicNotLast, node)
				err.Label = "This should be the last parameter"
				// Highlight the params after this
				err.AddHighlight("It should go after these", after)
				c.fileError(err, ctx.File)
			}
		}
	}
	if expr.ReturnType != nil {
		l.Return = c.parseType(expr.ReturnType, ctx)
	} else {
		l.Return = NothingType
	}
	l.Complete = true
	return l
}
