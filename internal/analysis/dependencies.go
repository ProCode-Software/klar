package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
)

type depMap map[string][]string

func getTypeDeps(t any) []string {
	var deps []string
	switch t := t.(type) {
	case []Type:
		for _, v := range t {
			deps = append(deps, getTypeDeps(v)...)
		}
	case []ast.TypePair:
		for _, v := range t {
			deps = append(deps, getTypeDeps(v.Value)...)
		}
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
	case ast.PrimitiveType:
		return nil
	default:
		panic(fmt.Sprintf("getTypeDeps: unhandled type %T", t))
	}
	return deps
}

type cycleError struct {
	dep, base string
	error     bool
}

func getAllDeps(typeDeps depMap, dep, base string) ([]string, cycleError) {
	depsOfDep := typeDeps[dep]
	if len(depsOfDep) == 0 {
		return nil, cycleError{}
	}
	list := make([]string, 0, len(depsOfDep))
	for _, dep := range depsOfDep {
		if dep == base {
			// Cycle
			return nil, cycleError{dep, base, true}
		}
		deps, err := getAllDeps(typeDeps, dep, base)
		if err.error {
			return nil, err
		}
		list = append(list, deps...)
	}
	return list, cycleError{}
}

func getTypeAliasDeps(types []ast.TypeAliasDeclaration) (depMap, cycleError) {
	var (
		typeDeps = make(map[string][]string, len(types))
		cycleErr cycleError
	)
	// Step 1: create list of all aliases each alias depends on
	for _, t := range types {
		var deps []string
		deps = append(deps, getTypeDeps(t.Type)...)
		typeDeps[t.Identifier] = deps
	}
	// Step 2: add the dependencies of those aliases
	// getAllDeps recursively adds deps
	for t, deps := range typeDeps {
		for _, dep := range deps {
			d, err := getAllDeps(typeDeps, dep, t)
			if err.error {
				cycleErr = err
				continue
			}
			typeDeps[t] = append(typeDeps[t], d...)
		}
	}
	return typeDeps, cycleErr
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
func sortTypeAliases(
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
