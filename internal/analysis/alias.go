package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

type TypeAlias struct {
	Name string
	Type
	resolved Type
}

func (a *TypeAlias) Resolve() Type {
	if a.resolved == nil {
		if subAlias, ok := a.Type.(*TypeAlias); ok {
			// This also recursively resolves aliases for upstream type aliases
			a.resolved = subAlias.Resolve()
		} else {
			// a's underlying type isn't an alias, so it is already resolved
			a.resolved = a.Type
		}
	}
	return a.resolved
}

func (a *TypeAlias) Kind() Kind       { return a.Resolve().Kind() }
func (a *TypeAlias) Underlying() Type { return a.Resolve() }

func (c *Checker) checkTypeAlias(o *Object, node *ast.TypeAliasDeclaration, fctx *Context) {
	tn := o.typ.(*TypeName)
	alias := &TypeAlias{Name: o.name}
	tn.Type = alias
	rhs := c.parseType(node.Type, fctx)
	// Set to invalid if we couldn't typecheck the rhs
	if rhs == nil {
		rhs = InvalidType
	}
	if rhs.Kind() == KindGeneric {
		// The target of a type alias cannot be a generic type
		err := klarerrs.Range(klarerrs.ErrGenericTypeAlias, node.Type.GetRange())
		err.Label = "This can't be a generic type"
		c.fileError(err, o.file)
		rhs = InvalidType
	}
	alias.Type = rhs
}

func (c *Checker) checkFuncAlias(o *Object) {
	decl := c.moduleDecls[o]
	targetExpr := decl.node.(*ast.FuncAliasDeclaration).Target
	// TODO: Lookup the target expression and make sure it resolves to a function
	var target *Object = nil

	_ = targetExpr
	o.typ.(*FunctionAlias).Target = target
}

func Unalias(t Type) Type {
	if a, ok := t.(*TypeAlias); ok {
		return a.Resolve()
	}
	return t
}
