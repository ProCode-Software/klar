package analysis

import (
	"fmt"
	"maps"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type depMap map[string][]string

const selfDep = "<self>"

func getTypeDeps(t any) []string {
	var deps []string
	switch t := t.(type) {
	case []ast.Type:
		deps = make([]string, 0, len(t))
		for _, v := range t {
			deps = append(deps, getTypeDeps(v)...)
		}
	case []ast.TypePair:
		deps = make([]string, 0, len(t))
		for _, v := range t {
			deps = append(deps, getTypeDeps(v.Value)...)
		}
	case nil:
		return nil
	case ast.Type:
		aliases := ast.CollectTypeAliases(t)
		deps = make([]string, 0, len(aliases))
		for _, alias := range aliases {
			deps = append(deps, alias.Identifier)
		}
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
	typeDeps := make(depMap, len(types))
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
	case ast.PrimitiveType, ast.BadExpression:
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
	var (
		intfDeps    = make(map[string][2][]string, len(intfs))
		allIntfDeps = make(depMap, len(intfs))
	)
	// Step 1 - get direct dependencies. categorize into c1 and c2
	for _, t := range intfs {
		var (
			name = t.Name()
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
		intfDeps[name] = [2][]string{c1Deps, c2Deps}
		// Preallocate full list
		allIntfDeps[name] = make([]string, 0, len(c1Deps)+len(c2Deps))
	}
	// Step 2 - copy direct interface deps -> aliases
	maps.Copy(aliases, allIntfDeps)
	// Step 3 - loop over interfaces
	for name, both := range intfDeps {
		// c1, c2
		for l, deps := range both {
			for _, dep := range deps {
				// Append direct dependency
				allIntfDeps[name] = append(allIntfDeps[name], dep)

				c.typeDepMode = l + 1 // set mode

				allIntfDeps[name] = append(allIntfDeps[name],
					c.getAllDeps(aliases, dep, name, 0, ctx)...,
				)
				c.typeDepMode = 0 // reset mode
			}
		}
		// Reassign to all dependencies
		aliases[name] = allIntfDeps[name]
	}
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
) ([]ast.TypeDeclaration, []string, map[string]ast.TypeDeclaration) {
	var (
		total   = len(aliases) + len(intfs)
		list    = make([]string, 0, total)
		names   = make([]string, 0, total)
		final   = make([]ast.TypeDeclaration, 0, total)
		typeMap = make(map[string]ast.TypeDeclaration, total)
		undef   = make(map[string]ast.TypeDeclaration)
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
		checkDefined(id, deps, typeMap, undef)
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
	return final, names, undef
}

func checkDefined(
	id string,
	deps []string,
	typeMap map[string]ast.TypeDeclaration,
	undefMap map[string]ast.TypeDeclaration,
) {
	for _, dep := range deps {
		if _, ok := typeMap[dep]; ok {
			continue
		}
		if undefMap[dep] == nil {
			undefMap[dep] = typeMap[id]
		}
	}
}

func traceUndefined(name string, t ast.TypeDeclaration) (r ranges.Range) {
	for _, alias := range ast.CollectTypeAliases(t) {
		if alias.Identifier == name {
			return alias.Base().Range
		}
	}
	// Should never happen
	panic(fmt.Sprintf("traceUndefined: %s not found", name))
}
