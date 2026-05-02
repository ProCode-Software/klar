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
	attrs   any // TODO
}

// NewObject returns a new [Object] without context information.
func NewObject(
	name string, fid FileID, rang ranges.Range, mod *Module, typ Type,
) *Object {
	return &Object{name: name, module: mod, rang: rang, file: fid, typ: typ}
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

// String returns a human-readable representation of the object.
func (obj *Object) String() string { return TypeToString(obj.typ) }

// Path returns the name of the object with the full import path.
func (obj *Object) Path() string {
	// TODO: should use '/' instead?
	return obj.Module().ImportPathString() + "/" + obj.name
}

// IsTypeDecl reports whether o represents a type declaration, including a type alias.
func (o *Object) IsTypeDecl() bool {
	_, ok := o.typ.(*TypeName)
	return ok
}

// FileName returns the base name of the file o was declared in.
func (o *Object) FileName() string { return o.module.ResolveFile(o.file) }

// FileName returns the full path of the file o was declared in.
func (o *Object) FilePath() string { return o.module.ResolveFilePath(o.file) }

// FileRange returns a [ranges.FileRange] representing the range of o's declaration
// and the base name of the containing file.
func (o *Object) FileRange() ranges.FileRange {
	return ranges.FileRange{o.rang, o.FileName()}
}

// FilePathRange returns a [ranges.FilePathRange] representing the range of o's
// declaration and the full path of the containing file.
func (o *Object) FilePathRange() ranges.FileRange {
	return ranges.FileRange{o.rang, o.FilePath()}
}

// Type Kinds
// ============

// Kind represents the kind of an object.
type Kind int

const (
	// Kinds that can be used as standalone [Type]s.
	InvalidType Kind = iota
	IntType
	StringType
	BoolType
	FloatType
	AnyType
	ErrorType
	NothingType

	KindList
	KindMap
	KindResult
	KindFunction
	KindUnion
	KindOptional

	KindEnum
	KindStruct
	KindInterface
	KindTag
	KindModule

	KindGeneric
	KindUnreachable // Nothing
)

// Kind returns the receiver.
func (k Kind) Kind() Kind { return k }

// String returns the name of the type k represents if k is a builtin.
func (k Kind) String() string { return "" } // TODO

// StringWithName implements [Type] and is equivalent to k.String()
func (k Kind) StringWithName(string) string { return k.String() }

// Types
// ==========

// Type represents a Klar object type or data type.
type Type interface {
	// Kind returns the kind of the type.
	Kind() Kind
	// String returns a human-readable string representation of the type
	// without a name.
	// String() string
	// StringWithName returns a human-readable string representation
	// of the type with the given name.
	// StringWithName(string) string
}

// TypeName represents a type declaration.
//
// Type is one of these types:
//   - [*TypeAlias]
//   - [*Struct]
//   - [*Interface]
//   - [*Enum]
//   - [*TagType]
type TypeName struct {
	Type
	Name string
}

// String returns the name of the type.
func (n *TypeName) String() string   { return n.Name }
func (n *TypeName) Underlying() Type { return n.Type }

// Underlyer is implemented by types or objects that have an underlying type.
type Underlyer interface {
	Type
	Underlying() Type
}

// MethodAdder is implemented by types that can have methods added to them.
// Per the spec, this is implemented by [*Struct] and [*Enum].
type MethodAdder interface {
	// Method returns the method with the given name, or nil if it doesn't exist.
	// Method(name string) *Function

	// AddMethod adds the method m to the type. If a method with the same name
	// already exists on the type, the existing method is returned instead.
	// m should have type [*Function], however, existing's type may not be [*Function].
	AddMethod(m *Object) (existing *Object)
}
