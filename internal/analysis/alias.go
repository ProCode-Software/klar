package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type TypeAlias struct {
	Type
	resolved Type
}

func (a *TypeAlias) Resolve() Type {
	if a.resolved != nil {
		return a.resolved
	}
	return nil
}

func (a *TypeAlias) Kind() Kind       { return a.Resolve().Kind() }
func (a *TypeAlias) Underlying() Type { return a.Resolve() }

func (c *Checker) checkTypeAlias(
	o *Object, node *ast.TypeAliasDeclaration, fileCtx *Context,
) {
	rhs := c.parseType(node.Type, fileCtx)
	o.typ.(*TypeName).Type = &TypeAlias{Type: rhs}
}

// TODO
func (c *Checker) resolveFuncAlias(fa *Object) {
}

func Unalias(t Type) Type {
	if a, ok := t.(*TypeAlias); ok {
		return a.Resolve()
	}
	return t
}
