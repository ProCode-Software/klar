package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/types"
)

type (
	depMap        map[string][]string
	groupedDepMap map[string][2][]string
)

const selfDep = "<self>"

func getC1AndC2Deps(typ ast.Type, c1Arr, c2Arr *[]string) {
	var list []ast.Type
	switch t := typ.(type) {
	case *ast.GenericType:
		list = append(list, t.Parameters...)
		list = append(list, t.Name)
	case *ast.ListType:
		getC1AndC2Deps(t.Value, c2Arr, c2Arr)
	case *ast.OptionalType:
		getC1AndC2Deps(t.Value, c2Arr, c2Arr)
	case *ast.UnionType:
		for _, opt := range t.Options {
			getC1AndC2Deps(opt, c2Arr, c2Arr)
		}
	case *ast.FunctionType:
		// list = append(list, t.Parameters...)
		list = append(list, t.ReturnType)
	case *ast.RestType:
		list = append(list, t.Value)
	case *ast.TupleType:
		list = append(list, t.Values...)
	case *ast.TypeAlias:
		*c1Arr = append(*c1Arr, t.Identifier)
	case *ast.MethodType:
		for _, param := range t.Parameters {
			list = append(list, param.Type)
		}
		list = append(list, t.ReturnType)
	case *ast.PrimitiveType, *ast.BadExpression:
		break
	default:
		panic(fmt.Sprintf("getC1AndC2Deps: unhandled type %T", t))
	}
	for _, t := range list {
		getC1AndC2Deps(t, c1Arr, c2Arr)
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

Iterate over each alias in the map (indefinite order) and add all dependencies
to a list. For the final list, loop over the list backwards and insert non-duplicates
for the final order.
*/
func sortDeps(depMap groupedDepMap) []string {
	var (
		total        = len(depMap)
		list         = make([]string, 0, total) // List that we're adding to
		final        = make([]string, total)
		alreadyAdded = make(map[string]bool, total)
	)
	// Add all dependencies into a flat list
	for id, both := range depMap {
		list = append(list, id) // Append self
		for _, group := range both {
			list = append(list, group...) // Append others
		}
	}
	// Loop backwards for the final order
	for i := len(list) - 1; i >= 0; i-- {
		if len(final) == total {
			return final
		}
		name := list[i]
		if alreadyAdded[name] {
			continue
		}
		final = append(final, name)
		alreadyAdded[name] = true
	}
	return final
}

type stackItem struct {
	Name string
	Pos  ranges.Range
}

func (c *Checker) SortTypes(names map[string]ast.TypeDeclaration, ctx context) []string {
	deps := make(groupedDepMap, len(names))
	allDeps := make(depMap, len(names))
	collectInherited := func(list []ast.Type, c1 *[]string) {
		for _, obj := range list {
			*c1 = append(*c1, obj.(*ast.TypeAlias).Identifier)
		}
	}
	// Get direct references
	for name, node := range names {
		var c1, c2 []string
		switch node := node.(type) {
		case *ast.StructDeclaration:
			collectInherited(node.InheritedTypes, &c1)
			for _, field := range node.Fields {
				getC1AndC2Deps(field.Type, &c1, &c2)
			}
		case *ast.InterfaceDeclaration:
			collectInherited(node.InheritedTypes, &c1)
			for _, field := range node.Fields {
				getC1AndC2Deps(field.Value, &c1, &c2)
			}
		case *ast.EnumDeclaration:
			collectInherited(node.Inherited, &c1)
			for _, item := range node.Values {
				for _, param := range item.Parameters {
					getC1AndC2Deps(param, &c1, &c1)
				}
			}
		case *ast.TypeAliasDeclaration:
			getC1AndC2Deps(node.Type, &c1, &c1)
		}
		deps[name] = [2][]string{c1, c2}
	}
	// Add all subdependencies
	for name, both := range deps {
		for mode, group := range both {
			for _, depInGroup := range group {
				// If depInGroup == name, it is directly referencing itself
				stack := []string{name}
				allDeps[name] = append(allDeps[name], c.getAllDeps(
					deps,        // Dependency map
					names[name], // Declaration node
					&stack,      // Default stack
					depInGroup,  // Current direct dependencies
					name, mode, ctx,
				)...)
			}
		}
	}
	return sortDeps(deps)
}

// Mode:
// 0 - non-struct without recursion;
// 1 - struct without recursion;
// 2 - struct with recursion
func (c *Checker) getAllDeps(
	depMap groupedDepMap,
	decl ast.TypeDeclaration,
	stack *[]string, // For tracking cycles
	dep,
	base string, // Type that we're getting all dependencies for
	mode int, // If set to 2, recursion is allowed
	ctx context,
) []string {
	modeIndex := max(0, mode-1)
	depsOfDep := depMap[dep][modeIndex]
	if len(depsOfDep) == 0 {
		return nil
	}
	*stack = append(*stack, dep)
	list := make([]string, 0, len(depsOfDep))
	for _, depOfDep := range depsOfDep {
		if depOfDep == base {
			// Cycle allowed in structs as array, optional, or union type
			if mode == 2 {
				list = append(list, selfDep)
				continue
			}
			// Cycle
			c.cycleErr(depMap, *stack, mode, decl, ctx)
			return nil
		}
		list = append(list, c.getAllDeps(
			depMap, decl, stack, depOfDep, base, mode, ctx,
		)...)
	}
	return list
}

// TODO: trace the cycle
// Error: Type cycle
// A references B here
// B references A here
func (c *Checker) cycleErr(
	depMap groupedDepMap, stack []string, mode int, decl ast.TypeDeclaration, ctx context,
) {
	for _, stackItem := range stack {
		ctx.DeclareType(stackItem, types.InvalidType, ranges.Range{})
	}
	err := errors.TypeError{
		Code: errors.ErrTypeCycle,
		Range:     decl.GetRange(),
		Params: errors.ErrorParams{
			"mode":  mode,
			"stack": stack,
		},
	}
	if mode == 1 {
		err.Hint("Structs and interfaces may refer to themselves inside list, optional, or union types.")
	}
	c.Error(err)
}
