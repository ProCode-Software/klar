package ast

import "fmt"

var stop bool

func Inspect(t Node, pred func(Node) bool) {
	if t == nil || pred == nil {
		return
	}
	Walk(t, func(t Node) {
		stop = !pred(t)
	})
}

func walkList[T Node](a []T, visitor func(Node)) {
	for _, t := range a {
		Walk(t, visitor)
	}
}

// Walk traverses a type AST in depth-first order. It first calls visitor
// on t, the on each child [Node] node.
func Walk(t Node, visitor func(Node)) {
	if stop || t == nil || visitor == nil {
		return
	}
	visitor(t)
	switch t := t.(type) {
	case *OptionalType:
		Walk(t.Value, visitor)
	case *ListType:
		Walk(t.Value, visitor)
	case *TupleType:
		walkList(t.Values, visitor)
	case *UnionType:
		walkList(t.Options, visitor)
	case *FunctionType:
		walkList(t.Parameters, visitor)
		Walk(t.ReturnType, visitor)
	case *GenericType:
		Walk(t.Name, visitor)
		walkList(t.Parameters, visitor)
	case *MethodType:
		for _, p := range t.Parameters {
			Walk(p.Type, visitor)
		}
		Walk(t.ReturnType, visitor)
	case *TypeAlias, *PrimitiveType, *BadExpression:
		return
	case *RestType:
		Walk(t.Value, visitor)
	case *TypeAliasDeclaration:
		Walk(t.Type, visitor)
	case *StructDeclaration:
		walkList(t.InheritedTypes, visitor)
		walkList(t.Fields, visitor)
	case *InterfaceDeclaration:
		walkList(t.InheritedTypes, visitor)
		walkList(t.Fields, visitor)
	case *StructField:
		Walk(t.Type, visitor)
		Walk(t.Value, visitor)
	case *TypePair:
		Walk(t.Value, visitor)
	default:
		panic(fmt.Sprintf("Walk: unhandled type %T", t))
	}
}

func WalkAll(n Node) (nodes []Node) {
	Walk(n, func(t Node) {
		nodes = append(nodes, t)
	})
	return nodes
}

// CollectTypeNames walks t, returning all [TypeAlias] and [PrimitiveType]
func CollectTypeNames(t Node) (types []Type) {
	Walk(t, func(t Node) {
		switch t := t.(type) {
		case *TypeAlias:
			types = append(types, t)
		case *PrimitiveType:
			types = append(types, t)
		}
	})
	return types
}

// CollectTypeAliases walks t, returning all [TypeAlias].
func CollectTypeAliases(t Node) (types []*TypeAlias) {
	Walk(t, func(t Node) {
		if t, ok := t.(*TypeAlias); ok {
			types = append(types, t)
		}
	})
	return types
}
