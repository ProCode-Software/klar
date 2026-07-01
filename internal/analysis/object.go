package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
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
	typ     ObjectKind
	order   uint32
	flags   Flag
	attrs   *Attributes
	info    *DeclarationInfo
}

// NewObject returns a new [Object] without context information.
func NewObject(
	name string, fid FileID, rang ranges.Range, mod *Module, typ ObjectKind,
) *Object {
	return &Object{name: name, module: mod, rang: rang, file: fid, typ: typ}
}

// Name returns the name of the object as declared in its module
func (obj *Object) Name() string { return obj.name }

// Context returns the context in which the object was declared
func (obj *Object) Context() *Context { return obj.context }

// Order returns the order in which the object was declared in the module.
func (obj *Object) Order() int { return int(obj.order) }

// FileContext returns the context for the file in which the object was declared.
// The return value is not equal to [Object.Context], but is the context where
// imported objects (that the object could depend on) are declared.
func (obj *Object) FileContext() *Context { return obj.module.fileContext[obj.file] }

// LookupContext returns the context in which imported objects (that the object
// could depend on) are declared. This is the object's file context, unless
// the object is declared in a nested scope (such as a function).
func (obj *Object) LookupContext() *Context {
	fctx := obj.FileContext()
	if obj.context.File <= 0 {
		return fctx
	}
	return obj.context
}

// Range returns the position of the object in the source code
func (obj *Object) Range() ranges.Range { return obj.rang }

// File returns the ID of the file in which the object was declared
func (obj *Object) File() FileID { return obj.file }

// Flags returns the flags applied to obj.
func (obj *Object) Flags() Flag { return obj.flags }

// Public returns whether the object is exported
func (obj *Object) Public() bool { return obj.public }

// Module returns the module in which the object was declared
func (obj *Object) Module() *Module { return obj.module }

// Type returns the type of the object
func (obj *Object) Type() ObjectKind { return obj.typ }

// Underlying is equivalent to obj.Type()
func (obj *Object) Underlying() Type { return obj.typ }

// Kind returns the kind of the object. Kind is equivalent to obj.Type().Kind().
func (obj *Object) Kind() Kind { return obj.typ.Kind() }

// String returns a human-readable representation of the object.
func (obj *Object) String() string { return obj.typ.String() }

// Path returns the name of the object with the full import path.
func (obj *Object) Path() string {
	// TODO: should use '/' instead?
	return obj.Module().ImportPathString() + "/" + obj.name
}

// TODO: If the object is top-level, these don't return a file. Find a solution

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

// IsTypeName reports whether o represents a type declaration.
func (o *Object) IsTypeName() bool {
	if o == nil {
		return false
	}
	_, ok := o.typ.(*TypeName)
	return ok
}

// TypeName returns o's Type() as a [*TypeName], or panics if
// o is not a type name.
func (o *Object) TypeName() *TypeName { return o.typ.(*TypeName) }

type ObjectKind interface {
	Type
	objKind()
	Underlying() Type
}

type InvalidTypeObject struct{}

func (o *InvalidTypeObject) Kind() Kind       { return InvalidType }
func (o *InvalidTypeObject) String() string   { return o.Kind().String() }
func (o *InvalidTypeObject) Underlying() Type { return o.Kind() }
func (o *InvalidTypeObject) objKind()         {}

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
	RegExType

	KindList
	KindMap
	KindResult
	KindFunction
	KindUnion
	KindOptional
	KindTuple
	KindTask

	KindEnum
	KindStruct
	KindInterface
	KindTag
	KindNamespace

	KindGeneric
)

// Kind returns the receiver. It panics if the receiver isn't a primitive.
func (k Kind) Kind() Kind {
	if !k.IsPrimitive() {
		panic(fmt.Sprintf("kind %d is not a primitive", k))
	}
	return k
}

func (k Kind) IsPrimitive() bool {
	switch k {
	case InvalidType, IntType, StringType, BoolType, FloatType,
		AnyType, ErrorType, NothingType, RegExType:
		return true
	default:
		return false
	}
}

// String returns the kind of the type as a human-readable string. If k is a
// primitive, the name of the Klar type is returned.
func (k Kind) String() string {
	return [...]string{
		// Primitives
		IntType:     "Int",
		StringType:  "String",
		BoolType:    "Bool",
		FloatType:   "Float",
		AnyType:     "Any",
		ErrorType:   "Error",
		NothingType: "Nothing",
		RegExType:   "RegEx",

		InvalidType:   "invalid type",
		KindList:      "list",
		KindMap:       "map",
		KindResult:    "Result",
		KindFunction:  "function",
		KindUnion:     "union",
		KindOptional:  "optional",
		KindTuple:     "tuple",
		KindTask:      "Task",
		KindEnum:      "enum",
		KindStruct:    "struct",
		KindInterface: "interface",
		KindTag:       "tag",
		KindNamespace: "module",
		KindGeneric:   "generic",
	}[k]
}

func (k Kind) IndexDot(i string) (Type, *klarerrs.Error) {
	if !k.IsPrimitive() {
		panic("cannot Index non-primitive type")
	}
	return indexBuiltin(k.String(), i)
}

func (k Kind) Index(i Type) (Type, *klarerrs.Error) {
	if !k.IsPrimitive() {
		panic("cannot Index non-primitive type")
	}
	if k != StringType {
		return noComputedIndex{}.Index(i)
	}

	// String is the only primitive that allows computed indexing
	if i.Kind() != IntType {
		return nil, indexTypeMismatchError(
			klarerrs.ErrNonNumericIndex,
			StringType, i, "Can't index String using type "+i.String(),
		)
	}
	// TODO: constant analysis (negative index, out of range index)
	return StringType, nil
}

// Types
// ==========

// Type represents a Klar object type or data type.
type Type interface {
	// Kind returns the kind of the type.
	Kind() Kind
	// String returns a human-readable string representation of the type
	// without a name.
	String() string
	// StringWithName returns a human-readable string representation
	// of the type with the given name.
	// StringWithName(string) string
}

// The result of a function call that doesn't return. Statements
// following this are unreachable.
type NoReturn struct{ Type }

func (u *NoReturn) Underlying() Type { return u.Type }

// Underlyer is implemented by types or objects that have an underlying type.
type Underlyer interface {
	Type
	// Returns the direct underlying type of the object.
	Underlying() Type
}

func Underlying(t Type) Type {
	for {
		oldT := t
		if u, ok := t.(Underlyer); ok {
			t = u.Underlying()
		} else {
			return t
		}
		// Tuples can't be compared, but if we reach this, there
		// is no underlying type.
		if _, ok := t.(Tuple); ok || t == oldT {
			return t
		}
	}
}

func UnderlyingTypeName(t Type) Type {
	for {
		oldT := t
		if u, ok := t.(Underlyer); ok {
			t = u.Underlying()
		} else {
			return t
		}
		// Tuples can't be compared, but if we reach this, there
		// is no underlying type.
		if _, ok := t.(Tuple); ok || t == oldT {
			return t
		}
		if _, ok := oldT.(*TypeName); ok {
			if _, ok := t.(Underlyer); !ok {
				return oldT
			}
		}
	}
}

type Untyped Kind

func (u Untyped) String() string {
	switch u {
	case Untyped(KindOptional):
		return "nil"
	case Untyped(KindList):
		return "[]"
	case Untyped(IntType):
		return "Int"
	case Untyped(KindStruct):
		return "struct"
	case Untyped(KindEnum):
		return "enum"
	default:
		panic(fmt.Sprintf("invalid untyped type: %s", Kind(u)))
	}
}

func (u Untyped) Kind() Kind { return Kind(u) }

// Used for shorthand struct/enum initialization.
type UntypedInit struct {
	Name   string
	kind   Kind // KindEnum or KindStruct
	Params []UntypedParam
}

func (i UntypedInit) Kind() Kind { return i.kind }

type UntypedParam struct {
	Label string
	Expr  ast.Expression
}

// Types that can be indexed (via `obj.index` or `obj[index]`) implement Indexer.
// If Index or IndexDot return (nil, nil), the type can't be indexed.
type Indexer interface {
	// Most types support IndexDot
	IndexDot(index string) (Type, *klarerrs.Error)
	// The following types support Index (won't return an error):
	// 	- [Map] when index is type [Map.Key]
	// 	- [List] when index is [IntType]
	// 	- [StringType] when index is [IntType]
	//  - [Tuple] when index is a constant [IntType]
	// Calling Index on any other type will return an error.
	Index(index Type) (Type, *klarerrs.Error)
}

type noComputedIndex struct{}

func (noComputedIndex) Index(index Type) (Type, *klarerrs.Error) {
	return nil, indexError(klarerrs.ErrInvalidComputedIndex, index, "")
}
