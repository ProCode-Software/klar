package analysis

import (
	"fmt"
	"maps"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/types"
)

type depMap map[string][]string

func getTypeDeps(t any) []string {
	var deps []string
	switch t := t.(type) {
	case []ast.Type:
		for _, v := range t {
			deps = append(deps, getTypeDeps(v)...)
		}
	case []ast.TypePair:
		for _, v := range t {
			deps = append(deps, getTypeDeps(v.Value)...)
		}
	case ast.MethodType:
		for _, v := range t.Parameters {
			deps = append(deps, getTypeDeps(v.Type)...)
		}
		deps = append(deps, getTypeDeps(t.ReturnType)...)
	case ast.TypePair:
		return getTypeDeps(t.Value)
	case ast.RestType:
		return getTypeDeps(t.Value)
	case ast.ListType:
		return getTypeDeps(t.Value)
	case ast.OptionalType:
		return getTypeDeps(t.Value)
	case ast.TupleType:
		return getTypeDeps(t.Values)
	case ast.FunctionType:
		deps = append(deps, getTypeDeps(t.Parameters)...)
		deps = append(deps, getTypeDeps(t.ReturnType)...)
	case ast.GenericType:
		deps = append(deps, getTypeDeps(t.Name)...)
		deps = append(deps, getTypeDeps(t.Parameters)...)
	case ast.UnionType:
		return getTypeDeps(t.Options)
	case ast.TypeAlias:
		return []string{t.Identifier}
	case ast.PrimitiveType, nil:
		return nil
	default:
		panic(fmt.Sprintf("getTypeDeps: unhandled type %T", t))
	}
	return deps
}

func (c *Checker) getAllDeps(typeDeps depMap, dep, base string, ctx *Context) []string {
	depsOfDep := typeDeps[dep]
	if len(depsOfDep) == 0 {
		return nil
	}
	list := make([]string, 0, len(depsOfDep))
	for _, dep := range depsOfDep {
		if dep == base {
			// Cycle
			ctx.SetType(base, types.InvalidType)
			ctx.SetType(dep, types.InvalidType)
			c.Error(errors.TypeError{
				ErrorCode: errors.ErrTypeCycle,
				Range:     ctx.TypeDeclarations[dep].Position,
			})
			return nil
		}
		list = append(list, c.getAllDeps(typeDeps, dep, base, ctx)...)
	}
	return list
}

func (c *Checker) getTypeAliasDeps(
	types []ast.TypeAliasDeclaration, ctx *Context,
) depMap {
	typeDeps := make(map[string][]string, len(types))
	// Step 1: create list of all aliases each alias depends on
	for _, t := range types {
		typeDeps[t.Identifier] = getTypeDeps(t.Type)
	}
	// Step 2: add the dependencies of those aliases
	// getAllDeps recursively adds deps
	for t, deps := range typeDeps {
		for _, dep := range deps {
			typeDeps[t] = append(typeDeps[t], c.getAllDeps(typeDeps, dep, t, ctx)...)
		}
	}
	return typeDeps
}

func getC1AndC2Deps(
	typ ast.Type, c1Arr, c2Arr *[]string,
) {
	var list []ast.Type
	switch typ := typ.(type) {
	case ast.GenericType:
		list = append(list, typ.Parameters...)
		list = append(list, typ.Name)
	case ast.ListType:
		*c2Arr = append(*c2Arr, getTypeDeps(typ.Value)...)
	case ast.OptionalType:
		*c2Arr = append(*c2Arr, getTypeDeps(typ.Value)...)
	case ast.UnionType:
		*c2Arr = append(*c2Arr, getTypeDeps(typ.Options)...)
	case ast.FunctionType:
		list = append(list, typ.Parameters...)
		list = append(list, typ.ReturnType)
	case ast.RestType:
		list = append(list, typ.Value)
	case ast.TupleType:
		list = append(list, typ.Values...)
	case ast.TypeAlias:
		*c1Arr = append(*c1Arr, typ.Identifier)
	case ast.PrimitiveType:
	default:
		panic(fmt.Sprintf("getC1AndC2Deps: unhandled type %T", typ))
	}
	for _, t := range list {
		getC1AndC2Deps(t, c1Arr, c2Arr)
	}
}

func (c *Checker) mergeStructDeps(
	aliases depMap, intfs []ast.TypeDeclaration,
) depMap {
	intfDeps := make(map[string][]string, len(intfs))
	for _, t := range intfs {
		var (
			deps []string
			// c1: Generic, static, inherited dependencies
			// c2: List, optional, union deps: cycle allowed
			c1Deps, c2Deps []string
		)
		switch t := t.(type) {
		case ast.StructDeclaration:
			c1Deps = append(c1Deps, getTypeDeps(t.InheritedTypes)...)
			for _, f := range t.Fields {
				getC1AndC2Deps(f.Type, &c1Deps, &c2Deps)
			}
		case ast.InterfaceDeclaration:
			c1Deps = append(c1Deps, getTypeDeps(t.InheritedTypes)...)
			for _, f := range t.Fields {
				getC1AndC2Deps(f.Value, &c1Deps, &c2Deps)
			}
		}
		intfDeps[t.Name()] = deps
	}
	maps.Copy(aliases, intfDeps)
	return aliases
}

/*
Sort them in the order that they should be declared in so types can reference each other.
Throws an error if there is a cycle.

For this example:

	type A = B
	type B = C
	type C = D
	type D = Int

the order would be: D, C, B, A.
C should be declared first so it can be referenced by B, and B can be referenced by A.
*/
func sortTypeDecls(
	depMap depMap, types []ast.TypeAliasDeclaration,
) []ast.TypeAliasDeclaration {
	var (
		list     = make([]string, 0, len(types))
		final    = make([]ast.TypeAliasDeclaration, 0, len(types))
		aliasMap = make(map[string]ast.TypeAliasDeclaration, len(types))
	)
	// Create the alias map
	for _, t := range types {
		aliasMap[t.Identifier] = t
	}
	// Add all dependencies into a flat list
	for id, deps := range depMap {
		list = append(list, append([]string{id}, deps...)...)
	}
	// Loop backwards for the final order
	alreadyAdded := make(map[string]bool, len(list))
	for i := len(list) - 1; i >= 0; i-- {
		if len(final) == len(types) {
			break
		}
		name := list[i]
		if alreadyAdded[name] {
			continue
		}
		final = append(final, aliasMap[name])
		alreadyAdded[name] = true
	}
	return final
}
