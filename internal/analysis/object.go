package analysis

import (
	"cmp"
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

type InvalidObject struct{}

func (o *InvalidObject) Kind() Kind       { return InvalidType }
func (o *InvalidObject) String() string   { return o.Kind().String() }
func (o *InvalidObject) Underlying() Type { return o.Kind() }
func (o *InvalidObject) objKind()         {}

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

func (k Kind) Index(i string, t *Expr) *klarerrs.Error {
	if !k.IsPrimitive() {
		panic("cannot Index non-primitive type")
	}
	return indexBuiltin(k.String(), i, t)
}

func (k Kind) IndexComputed(i Type, t *Expr) *klarerrs.Error {
	switch {
	case !k.IsPrimitive():
		panic("cannot Index non-primitive type")
	case k != StringType:
		// String is the only primitive that allows computed indexing
		return indexError(klarerrs.ErrInvalidComputedIndex, i, "")
	case i.Kind() != IntType:
		return indexTypeMismatchError(
			klarerrs.ErrNonNumericIndex,
			StringType, i, "Can't index String using type "+i.String(),
		)
	default:
		// TODO: constant analysis (negative index, out of range index)
		t.Type = StringType
		return nil
	}
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

func As[T Type](t Type) T { return Underlying(t).(T) }

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

// Types that can be dot-indexed (via `obj.index`) implement Indexer.
// If Index sets t's Type to nil and returns nil, the type can't be indexed.
type Indexer interface {
	Index(index string, t *Expr) *klarerrs.Error
}

// Types can be indexed via `obj[index]`.
// ComputedIndexer is implemented by the following types:
//   - [Map] when index is type [Map.Key]
//   - [List] when index is [IntType]
//   - [StringType] when index is [IntType]
//   - [Tuple] when index is a constant [IntType]
type ComputedIndexer interface {
	IndexComputed(index Type, t *Expr) *klarerrs.Error
}

// Per the spec
var (
	_ = [...]ComputedIndexer{&Map{}, &List{}, StringType, Tuple{}}
	_ = [...]Indexer{
		&Map{}, &List{}, &Struct{}, &Enum{}, &Interface{}, &Task{},
		StringType, IntType, FloatType, ErrorType,
	}
)

// The result of a function call that doesn't return. Statements
// following this are unreachable.
type NoReturn struct{ Type }

func (nr *NoReturn) IsTODO() bool { return nr.Type == nil }

func (u *NoReturn) Underlying() Type { return cmp.Or[Type](u.Type, u /* is a TODO */) }

// Untyped can only be one of:
//   - [KindOptional] (for nil literal)
//   - [KindList] (for empty list literal)
//   - [KindMap] (for empty map literal)
//   - [IntType] (for numeric (non-float) literal).
type Untyped Kind

func (u Untyped) String() string {
	switch u {
	case Untyped(KindOptional):
		return "none"
	case Untyped(KindList):
		return "[]"
	case Untyped(IntType):
		return "Int"
	case Untyped(KindMap):
		return "#{}"
	default:
		panic(fmt.Sprintf("invalid untyped type: %s", Kind(u)))
	}
}

func (u Untyped) Kind() Kind { return Kind(u) }

// Used for shorthand struct/enum initialization.
type UntypedInit struct {
	kind   Kind           // [KindEnum] or [KindStruct]
	Node   ast.Expression // [*ast.EnumLiteral] or [*ast.StructDotInit]
	Params []*ast.CallParam
}

func (i *UntypedInit) Kind() Kind     { return i.kind }
func (i *UntypedInit) String() string { return i.kind.String() }

func (c *Checker) toTyped(typ, hint Type, node ast.Expression, fid FileID) Type {
	if hint != nil {
		return hint
	}
	switch ut := typ.(type) {
	case Untyped:
		switch ut.Kind() {
		case KindOptional:
			// Untyped nil
			err := klarerrs.Node(klarerrs.ErrUntypedNil, node)
			err.Label = "I don't know what optional type this is"
			c.fileError(err, fid)
			return InvalidType
		case IntType:
			// If untyped, then it's an Int by default
			return IntType
		case KindList:
			// No hint and no list items: unknown list type
			err := klarerrs.Node(klarerrs.ErrUntypedEmptyList, node)
			err.Label = "This list is empty and its type can't be inferred"

			// Suggest hints
			err.Hint("If you're declaring a variable, add a type annotation before ':='.")

			diff2 := klarerrs.NewDiff(
				c.module.ResolveFilePath(fid),
				klarerrs.AddedString{Position: node.GetRange().Start, String: "[T]("},
				klarerrs.AddedString{Position: node.GetRange().End, String: ")"},
			)
			err.HintWithDiff(
				"Otherwise, initialize an empty list with a specific type. (Replace 'T' with the intended item type)",
				diff2,
			)
			c.fileError(err, fid)
			return InvalidType
		case KindMap:
			// No hint and no map items: unknown map type
			err := klarerrs.Node(klarerrs.ErrUntypedEmptyMap, node)
			err.Label = "This map is empty and its type can't be inferred"
			// TODO: Diff
			c.fileError(err, fid)
			return InvalidType
		default:
			panic(fmt.Sprintf("unhandled Untyped type: Untyped(%s)", ut.Kind()))
		}
	case *UntypedInit:
		if enum, ok := ut.Node.(*ast.EnumLiteral); ok {
			err := klarerrs.Node(klarerrs.ErrUntypedEnum, enum)
			err.Label = "I don't know the type of this enum"
			diff := klarerrs.NewDiff(
				c.module.ResolveFilePath(fid),
				klarerrs.AddedString{Position: enum.Range.Start, String: "T"},
			)
			err.HintWithDiff(
				"Add an explicit type before the enum item. (Replace 'T' with the intended type)",
				diff,
			)
			c.fileError(err, fid)
			return InvalidType
		}
		// Struct
		err := klarerrs.Node(klarerrs.ErrUntypedStruct, ut.Node)
		err.Label = "I don't know the type of this struct"
		diff := klarerrs.NewDiff(
			c.module.ResolveFilePath(fid),
			klarerrs.DeletedRange{ranges.SingleChar(ut.Node.GetRange().Start)}, // '.'
			klarerrs.AddedString{Position: ut.Node.GetRange().Start, String: "T"},
		)
		err.HintWithDiff(
			"Add an explicit type before the parameters. (Replace 'T' with the intended type)",
			diff,
		)
		c.fileError(err, fid)
		return InvalidType
	default:
		return typ // Already typed (TODO: This could be list of untyped)
	}
}
