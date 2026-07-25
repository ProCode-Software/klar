package analysis

import (
	"cmp"
	"fmt"
	"slices"
)

// Compatible returns whether type a is compatible with b.
func Compatible(a, b Type) bool {
	// Allow checking for compatibility with kinds
	// Example: Compatible(t, KindStruct) = true if t is a struct
	if b, ok := b.(Kind); ok && !b.IsPrimitive() {
		return a.Kind() == b
	}
	aKind, bKind := a.Kind(), b.Kind()
	switch a := Underlying(a).(type) {
	case Untyped:
		// Untyped Int is compatible with Int and Float. For other untyped
		// types, they're compatible if they have the same Kind.
		return aKind == bKind || (a == Untyped(IntType) && bKind == FloatType)
	case *UntypedInit:
		return aKind == bKind
	case *NoReturn:
		if a.Type == nil { // Always true for *NoReturn to be an underlying type
			return true // This is a TODO function and is compatible with any type
		}
	}
	// If the target is a TODO, it will be assigned to Any?
	if nrb, ok := b.(*NoReturn); ok && nrb.Type == nil {
		b = &Optional{AnyType}
	}
	switch {
	case bKind == AnyType:
		// A => Any if A != nil. All non-optional types are compatible with Any
		return aKind != KindOptional
	case aKind == KindList && bKind == KindList:
		// [A] => [B] if A => B
		a = Underlying(a).(*List).Elem
		b = Underlying(b).(*List).Elem
		return Compatible(a, b)
	case bKind == KindOptional && aKind != KindOptional:
		// A => B? if A => B
		if b == Untyped(KindOptional) {
			return true // Non-optional to untyped nil
		}
		b := Underlying(b).(*Optional)
		return Compatible(a, b.Elem)
	case bKind == KindOptional && aKind == KindOptional:
		// A? => B? if A => B | nil
		a = Underlying(a).(*Optional).Elem
		b = Underlying(b).(*Optional).Elem
		return Compatible(a, b)
	case bKind == KindResult && aKind != KindResult:
		// A => Result<B, C> if A => B | C
		b := Underlying(b).(*Result)
		return Compatible(a, b.Success) || Compatible(a, b.Error)
	case bKind == KindUnion && aKind == KindResult:
		// Result<A, B> => C | D if A => C | D and B => C | D.
		//
		// We won't allow compatibility the other way around because the compiler
		// needs to know at compile-time which type is the error, and which is the
		// success type (assertions do exist).
		//
		// TODO: Should we keep Result => union compatibility? Results are intended
		// for error handling, and assigning to the union means not checking the error.
		ra := Underlying(a).(*Result)
		return Compatible(ra.Success, b) && Compatible(ra.Error, b)
	case bKind == KindTag, bKind == KindInterface:
		// TODO
		if Implements(a, b) {
			return true
		}
	case bKind == KindUnion && aKind != KindUnion:
		// A => B | C if A => B or A => C
		ub := As[*Union](b)
		return slices.ContainsFunc(ub.Types, func(opt Type) bool {
			return Compatible(a, opt)
		})
	case bKind == KindUnion && aKind == KindUnion:
		// AA | AB => BA | BB if (AA => BA | BB) and (AB => BA | BB)
		//
		// Check that each type in union A is compatible with the entire union B
		ua, ub := As[*Union](a), As[*Union](b)
		for _, ta := range ua.Types {
			if !slices.ContainsFunc(ub.Types, func(tb Type) bool {
				return Compatible(ta, tb)
			}) {
				return false
			}
		}
		return true
	case bKind == KindMap && aKind == KindMap:
		// #{KA: VA} => #{KB: VB} if KA => KB and VA => VB
		ma, mb := As[*Map](a), As[*Map](b)
		return Compatible(ma.Key, mb.Key) && Compatible(ma.Value, mb.Value)
	case bKind == KindEnum && aKind == KindEnum:
		ea := Underlying(a).(EnumParenter)
		eb := Underlying(b).(EnumParenter)
		return ea.EnumParent() == eb.EnumParent()
	}
	return TypesEqual(a, b) // TODO
}

// TypesEqual returns whether the underlying types of a and b are equal.
func TypesEqual(a, b Type) bool {
	a, b = Underlying(a), Underlying(b)
	// If one is invalid type, avoid showing many errors, so we will say
	// they are compatible.
	if a.Kind() == InvalidType || b.Kind() == InvalidType {
		return true
	}
	if a.Kind() != b.Kind() {
		return false
	}
	// If the kinds are the same, any untyped type is compatible
	switch b.(type) {
	case Untyped, *UntypedInit, Kind:
		return true
	}
	switch a := a.(type) {
	case Untyped, *UntypedInit, Kind:
		return true
	case *List:
		return TypesEqual(a.Elem, b.(*List).Elem)
	case *Optional:
		return TypesEqual(a.Elem, b.(*Optional).Elem)
	case *Map:
		b := b.(*Map)
		return TypesEqual(a.Key, b.Key) && TypesEqual(b.Value, b.Value)
	case *Tuple:
		return slices.EqualFunc(a.Items, b.(*Tuple).Items, TypesEqual)
	case *Result:
		b := b.(*Result)
		return TypesEqual(a.Success, b.Success) && TypesEqual(a.Error, b.Error)
	case *Task:
		return TypesEqual(a.Result, b.(*Task).Result)
	}
	// Such as structs, enums
	return a == b
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

func IsConcreteType(t Type) bool {
	switch t.Kind() {
	case KindUnion, KindInterface, KindTag, KindResult, KindOptional:
		return false
	}
	return true
}

// Implements returns whether type a implements b.
func Implements(a, b Type) bool {
	a = Underlying(a)
	if b.Kind() != KindTag {
		// TODO: Interface (and possibly struct) implementation checking
		return false
	}
	b = Underlying(b).(*Tag)
	var inherited map[Type]struct{}
	switch a := Underlying(a).(type) {
	case *Enum:
		inherited = a.Inherited
	case *Struct:
		inherited = a.Inherited
	case *Interface:
		inherited = a.Inherited
	case *Tag:
		if a == b {
			return true
		}
		inherited = a.Implements
	}
	for a := range inherited {
		if Compatible(a, b) {
			return true
		}
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

// CommonType returns the largest type of a and b. If a and b are compatible
// with each other, CommonTyp returns a. If neither are compatible, CommonType
// returns nil.
//
// Examples:
//
//	CommonType(UntypedInt, Int) -> Int
//	CommonType(interface A, type that implements A) -> A
func CommonType(a, b Type) Type {
	compatAB := Compatible(a, b)
	compatBA := Compatible(b, a)
	switch {
	case compatAB && !compatBA:
		return b
	case compatBA && !compatAB:
		return a
	case compatAB && compatBA:
		return a
	case !compatAB && !compatBA:
		return nil
	default:
		panic(fmt.Sprintf(
			"CommonType(a, b): unhandled: Compatible(a, b) = %t, Compatible(b, a) = %t",
			compatAB, compatBA,
		))
	}
}

// commonTypeOptional is [CommonType], but accepts nil arguments. If
// any argument is nil, the non-nil argument is returned. If both are nil,
// commonTypeOptional panics. commonTypeOptional returns nil if
// [CommonType](a, b) returns nil.
func commonTypeOptional(a, b Type) Type {
	if a == nil || b == nil {
		nonNil := cmp.Or(a, b)
		if nonNil == nil {
			panic("commonTypeOptional(nil, nil)")
		}
		return nonNil
	}
	return CommonType(a, b)
}

func IsUntyped(t Type) bool {
	switch Underlying(t).(type) {
	case Untyped, *UntypedInit:
		return true
	default:
		return false
	}
}
