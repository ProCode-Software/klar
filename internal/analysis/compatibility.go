package analysis

import (
	"reflect"
	"slices"
)

// Compatible returns whether type a is compatible with b.
func Compatible(a, b Type) bool {
	if ut, ok := a.(Untyped); ok {
		if ut == Untyped(IntType) && (b == IntType || b == FloatType) {
			return true
		}
		// Untyped nil and untyped empty list
		if Kind(ut) == b.Kind() {
			return true
		}
	}
	// Allow checking for compatibility with kinds
	// Example: Compatible(t, KindStuct) returns true if t is a struct
	if b, ok := b.(Kind); ok && !b.IsPrimitive() {
		return a.Kind() == b
	}
	if a, ok := a.(*NoReturn); ok && a.Type == nil {
		return true // This is a TODO function and is compatible with any type
	}
	if a.Kind() == KindList && b.Kind() == KindList {
		return Compatible(Underlying(a).(*List).Elem, Underlying(b).(*List).Elem)
	}
	if b.Kind() == KindOptional && a.Kind() != KindOptional {
		b = Underlying(b).(*Optional).Elem
		return Compatible(a, b)
	}
	if b.Kind() == KindResult && a.Kind() != KindResult {
		b := Underlying(b).(*Result)
		return Compatible(a, b.Success) || Compatible(a, b.Error)
	}
	return TypesEqual(a, b) // TODO
}

// TypesEqual returns whether the underlying types of a and b are equal.
func TypesEqual(a, b Type) bool {
	a, b = Underlying(a), Underlying(b)

	// Tuples can't be compared via '==' in Go
	if tupA, ok := a.(Tuple); ok {
		tupB, ok := b.(Tuple)
		if !ok {
			return false // One is a tuple and the other isn't
		}
		return slices.EqualFunc(tupA, tupB, TypesEqual)
	}
	return a == b || reflect.DeepEqual(a, b)
}

func ConcreteTypeOf(t Type) Type {
	t = Underlying(t)
	switch t := t.(type) {
	case *Optional:
		return t.Elem
	case *Result:
		return t.Success
	default:
		return t
	}
}
