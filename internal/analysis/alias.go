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

func (c *Checker) checkTypeAlias(o *Object, node *ast.TypeAliasDeclaration, fctx *Context) {
	rhs := c.parseType(node.Type, fctx)
	o.typ.(*TypeName).Type = &TypeAlias{Type: rhs}
}

func (c *Checker) resolveFuncAlias(fa *Object) {
	decl := c.moduleDecls[fa]
	targetExpr := decl.node.(*ast.FuncAliasDeclaration).Target
	// TODO: Lookup the target expression and make sure it resolves to a function
	var target *Object = nil
	_ = targetExpr
	fa.typ.(*FunctionAlias).Target = target
}

func Unalias(t Type) Type {
	if a, ok := t.(*TypeAlias); ok {
		return a.Resolve()
	}
	return t
}
