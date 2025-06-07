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

func getTypeDeclDeps(types []ast.TypeDeclaration) depMap {
	typeDeps := make(map[string][]string, len(types))
	for _, t := range types {
		var deps []string
		switch t := t.(type) {
		case ast.TypeAliasDeclaration:
			typeDeps[t.Identifier] = getTypeDeps(t.Type)
		case ast.StructDeclaration:
			deps = append(deps, getTypeDeps(t.InheritedTypes)...)
			for _, v := range t.Fields {
				deps = append(deps, getTypeDeps(v.Type)...)
			}
			typeDeps[t.Identifier] = deps
		case ast.InterfaceDeclaration:
			deps = append(deps, getTypeDeps(t.InheritedTypes)...)
			deps = append(deps, getTypeDeps(t.Fields)...)
			typeDeps[t.Identifier] = deps
		}
	}
	return typeDeps
}

/*
Sort them in the order that they should be declared in so types can reference each other.
Throws an error if there is a cycle.

We can sort them by iterating over each dependency and adding it to the top of the stack. Or
the bottom of the stack and reverse it at the end.

For this example:

	type A = B
	type B = C
	type C = Int

the order would be: C, B, A.
C should be declared first so it can be referenced by B, and B can be referenced by A.

If there is a cycle:

	type A = B
	type B = C
	type C = A
*/
func sortTypeDeclDeps(deps depMap, types []ast.TypeDeclaration) []ast.TypeDeclaration {
}
