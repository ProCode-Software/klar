package analysis

import (
	"reflect"
	"slices"
)

// Compatible returns whether type a is compatible with b.
func Compatible(a, b Type) bool {
	aKind, bKind := a.Kind(), b.Kind()
	if ut, ok := a.(Untyped); ok {
		if ut == Untyped(IntType) && (b == IntType || b == FloatType) {
			return true
		}
		// Untyped nil and untyped empty list
		if Kind(ut) == bKind {
			return true
		}
	}
	// Allow checking for compatibility with kinds
	// Example: Compatible(t, KindStuct) returns true if t is a struct
	if b, ok := b.(Kind); ok && !b.IsPrimitive() {
		return aKind == b
	}
	if a, ok := a.(*NoReturn); ok && a.Type == nil {
		return true // This is a TODO function and is compatible with any type
	}
	switch {
	case aKind == KindList && bKind == KindList:
		a = Underlying(a).(*List).Elem
		b = Underlying(b).(*List).Elem
		return Compatible(a, b)
	case bKind == KindOptional && aKind != KindOptional:
		b = Underlying(b).(*Optional).Elem
		return Compatible(a, b)
	case bKind == KindResult && aKind != KindResult:
		b := Underlying(b).(*Result)
		return Compatible(a, b.Success) || Compatible(a, b.Error)
	case bKind == KindTag, bKind == KindInterface:
		if Implements(a, b) {
			return true
		}
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

// Implements returns whether type a implements b.
func Implements(a, b Type) bool {
	a = Underlying(a)
	if b.Kind() != KindTag {
		// TODO: Interface (and possibly struct) implementation checking
		return false
	}
	b = Underlying(b).(*Tag)
	switch a := Underlying(a).(type) {
	case *Enum:
		_, ok := a.Inherited[b]
		return ok
	case *Struct:
		_, ok := a.Inherited[b]
		return ok
	case *Interface:
		_, ok := a.Inherited[b]
		return ok
	case *Tag:
		_, ok := a.Implements[b]
		return ok
	}
	return false
}

func isTypeName(t Type) bool {
	switch t := t.(type) {
	case *TypeName:
		return true
	case *Object:
		return t.IsTypeName()
	default:
		return false
	}
}
