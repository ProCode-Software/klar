package analysis

import (
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Object represents the type of a Klar object.
type Object struct {
	name    string
	context *Context
	rang    ranges.Range
	file    FileID
	public  bool
	module  *Module
	typ     Type
	order   uint32
	flags   Flag
}

// Name returns the name of the object as declared in its module
func (obj *Object) Name() string { return obj.name }

// Context returns the context in which the object was declared
func (obj *Object) Context() *Context { return obj.context }

// Range returns the position of the object in the source code
func (obj *Object) Range() ranges.Range { return obj.rang }

// File returns the base name of the file in which the object was declared
func (obj *Object) File() FileID { return obj.file }

// Flags returns the flags applied to obj.
func (obj *Object) Flags() Flag { return obj.flags }

// Public returns whether the object is exported
func (obj *Object) Public() bool { return obj.public }

// Module returns the module in which the object was declared
func (obj *Object) Module() *Module { return obj.module }

// Type returns the type of the object
func (obj *Object) Type() Type { return obj.typ }

// Kind returns the kind of the object. Kind is equivalent to obj.Type().Kind().
func (obj *Object) Kind() Kind { return obj.typ.Kind() }

// String returns a human-readable representation of the object
func (obj *Object) String() string { return "" }

// Path returns the name of the object with the full import path.
func (obj *Object) Path() string {
	// TODO: should use '/' instead?
	return obj.Module().ImportPathString() + "." + obj.name
}

type Kind int

const (
	_ Kind = iota
	KindInt
	KindString
	KindBool
	KindFloat
	KindList
	KindMap
	KindResult
	KindAny
	KindFunction
	KindError
	KindNothing

	KindEnum
	KindStruct
	KindInterface
	KindTag
	KindUnion
	KindOptional
	KindFunctionAlias
	KindModule

	KindInvalid
	KindUnreachable // Nothing
)

type Type interface {
	Kind() Kind
	String() string
}

// Kind returns the receiver.
func (k Kind) Kind() Kind { return k }

// NewObject returns a new [Object] without context information.
func NewObject(
	name string, fid FileID, rang ranges.Range, mod *Module, typ Type,
) *Object {
	return &Object{
		name:   name,
		module: mod,
		rang:   rang,
		file:   fid,
		typ:    typ,
	}
}

type TypeName struct{Type}