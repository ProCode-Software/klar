package analysis

import (
	"fmt"
	"maps"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/types"
)

type depMap map[string][]string

const selfDep = "<self>"

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

func (c *Checker) getAllDeps(
	typeDeps depMap, dep, base string, level int, ctx *Context,
) []string {
	depsOfDep := typeDeps[dep]
	if len(depsOfDep) == 0 {
		return nil
	}
	list := make([]string, 0, len(depsOfDep))
	for _, dep2 := range depsOfDep {
		if dep2 == base {
			// Cycle allowed in structs as array, optional, or union type
			if c.typeDepMode == 2 {
				list = append(list, selfDep)
				continue
			}
			// Cycle
			ctx.SetType(base, types.InvalidType)
			ctx.SetType(dep2, types.InvalidType)
			err := errors.TypeError{
				ErrorCode: errors.ErrTypeCycle,
				Range:     ctx.TypeDeclarations[dep2].Position,
				Params: errors.ErrorParams{
					"mode":   c.typeDepMode,
					"isSelf": level == 0,
					"types":  [2]string{base, dep},
				},
			}
			if c.typeDepMode == 1 {
				err.Hint("Structs and interfaces may refer to themselves inside list, optional, or union types.")
			}
			c.Error(err)
			return nil
		}
		list = append(list, c.getAllDeps(typeDeps, dep2, base, level+1, ctx)...)
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
			typeDeps[t] = append(typeDeps[t], c.getAllDeps(typeDeps, dep, t, 0, ctx)...)
		}
	}
	return typeDeps
}

func getC1AndC2Deps(typ ast.Type, c1Arr, c2Arr *[]string) {
	var list []ast.Type
	switch t := typ.(type) {
	case ast.GenericType:
		list = append(list, t.Parameters...)
		list = append(list, t.Name)
	case ast.ListType:
		*c2Arr = append(*c2Arr, getTypeDeps(t.Value)...)
	case ast.OptionalType:
		*c2Arr = append(*c2Arr, getTypeDeps(t.Value)...)
	case ast.UnionType:
		*c2Arr = append(*c2Arr, getTypeDeps(t.Options)...)
	case ast.FunctionType:
		list = append(list, t.Parameters...)
		list = append(list, t.ReturnType)
	case ast.RestType:
		list = append(list, t.Value)
	case ast.TupleType:
		list = append(list, t.Values...)
	case ast.TypeAlias:
		*c1Arr = append(*c1Arr, t.Identifier)
	case ast.MethodType:
		for _, param := range t.Parameters {
			list = append(list, param.Type)
		}
		list = append(list, t.ReturnType)
	case ast.PrimitiveType:
		break
	default:
		panic(fmt.Sprintf("getC1AndC2Deps: unhandled type %T", t))
	}
	for _, t := range list {
		getC1AndC2Deps(t, c1Arr, c2Arr)
	}
}

func (c *Checker) mergeStructDeps(
	aliases depMap, intfs []ast.TypeDeclaration, ctx *Context,
) {
	intfDeps := make(map[string][]string, len(intfs))
	for _, t := range intfs {
		var (
			name = t.Name()
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
		deps = make([]string, 0, len(c1Deps)+len(c2Deps))
		deps = append(deps, c1Deps...)
		deps = append(deps, c2Deps...)
		for _, list := range [][]string{c1Deps, c2Deps} {
			for _, dep := range list {
				intfDeps[name] = append(intfDeps[name],
					c.getAllDeps(intfDeps, dep, name, 0, ctx)...)
			}
		}
		intfDeps[name] = deps
	}
	maps.Copy(aliases, intfDeps)
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
	depMap depMap,
	aliases []ast.TypeAliasDeclaration,
	intfs []ast.TypeDeclaration,
) ([]ast.TypeDeclaration, []string) {
	var (
		total   = len(aliases) + len(intfs)
		list    = make([]string, 0, total)
		names   = make([]string, 0, total)
		final   = make([]ast.TypeDeclaration, 0, total)
		typeMap = make(map[string]ast.TypeDeclaration, total)
	)
	// Create the map of types
	for _, t := range aliases {
		typeMap[t.Name()] = t
	}
	for _, t := range intfs {
		typeMap[t.Name()] = t
	}
	// Add all dependencies into a flat list
	for id, deps := range depMap {
		list = append(list, append([]string{id}, deps...)...)
	}
	// Loop backwards for the final order
	alreadyAdded := make(map[string]bool, len(list))
	for i := len(list) - 1; i >= 0; i-- {
		if len(final) == total {
			break
		}
		name := list[i]
		if alreadyAdded[name] {
			continue
		}
		final = append(final, typeMap[name])
		names = append(names, name)
		alreadyAdded[name] = true
	}
	return final, names
}
