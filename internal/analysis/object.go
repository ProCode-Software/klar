package analysis

import (
	"strings"

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

// String returns a human-readable representation of the object
func (obj *Object) String() string {
	// TODO
	return obj.typ.StringWithName(obj.name)
}

func (obj *Object) StringWithName(name string) string {
	return obj.typ.StringWithName(name) // TODO
}

// Path returns the name of the object with the full import path.
func (obj *Object) Path() string {
	// TODO: should use '/' instead?
	return obj.Module().ImportPathString() + "." + obj.name
}

// IsTypeDecl reports whether o represents a type declaration.
func (o *Object) IsTypeDecl() bool {
	_, ok := o.typ.(*TypeName)
	return ok
}

// FileName returns the base name of the file o was declared in.
func (o *Object) FileName() string {
	return o.module.ResolveFile(o.file)
}

// FileName returns the full path of the file o was declared in.
func (o *Object) FilePath() string {
	return o.module.ResolveFilePath(o.file)
}

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
	KindInvalid Kind = iota
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

type Type interface {
	// Kind returns the kind of the type.
	Kind() Kind
	// String returns a human-readable string representation of the type
	// without a name.
	String() string
	// StringWithName returns a human-readable string representation
	// of the type with the given name.
	StringWithName(string) string
}

// TypeName represents a type declaration.
type TypeName struct {
	Underlying Type
	Name       string
}

func (n *TypeName) String() string                    { return n.Name }
func (n *TypeName) StringWithName(name string) string { return name }
func (n *TypeName) Kind() Kind                        { return n.Underlying.Kind() }

// Function represents a function type, either a declared function or a lambda.
type Function struct {
	Self      *Variable // If method
	Overloads []*Overload
	Return    Type
}

func (fn *Function) Kind() Kind     { return KindFunction }
func (fn *Function) String() string { return fn.StringWithName("") }
func (fn *Function) StringWithName(name string) string {
	var b strings.Builder
	b.WriteString("func")
	if name != "" {
		b.WriteByte(' ')
		b.WriteString(name)
	}
	if len(fn.Overloads) == 1 {
		b.WriteString(fn.Overloads[0].String())
	}
	if fn.Return != nil {
		switch fn.Return.Kind() {
		case KindNothing, KindInvalid, KindUnreachable:
		default:
			b.WriteString(" -> ")
			b.WriteString(fn.Return.String())
		}
	}
	return b.String()
}

// TODO: params with defaults
type Overload struct {
	Generics       []*Generic
	Params         []*Variable
	LabelledParams []*LabelledParam
}

func (o *Overload) StringWithName(name string) string {
	return "func " + name + o.String()
}

func (o *Overload) String() string {
	var b strings.Builder
	// Generics
	if len(o.Generics) > 0 {
		b.WriteByte('<')
		for i, g := range o.Generics {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(g.Name)
		}
		b.WriteByte('>')
	}
	// Params
	b.WriteByte('(')
	for i, param := range o.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.String())
	}
	// Labelled params
	for i, param := range o.LabelledParams {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(param.String())
	}
	b.WriteByte(')')
	return b.String()
}

type LabelledParam struct {
	Name string
	Type Type
}

func (p *LabelledParam) String() string { return p.Name + ": " + p.Type.String() }

func (p *LabelledParam) StringWithName(string) string { return p.String() }

type FunctionAlias struct {
	Self   *Variable // If method
	Origin *Object
}

func (a *FunctionAlias) Kind() Kind { return KindFunction }

func (a *FunctionAlias) String() string {
	return "func = " + "" // TODO
}

func (a *FunctionAlias) StringWithName(name string) string {
	return "func " + name + " = " + ""
}

// Generic represents a generic type parameter.
type Generic struct{ Name string }

func (g *Generic) Kind() Kind                        { return KindGeneric }
func (g *Generic) String() string                    { return "<" + g.Name + ">" }
func (g *Generic) StringWithName(name string) string { return "<" + name + ">" }

// Variable represents a variable type.
type Variable struct {
	Name    string
	VarKind VariableKind
	Type
}

func (v *Variable) Underlying() Type { return v.Type }

type VariableKind uint8

const (
	_              VariableKind = iota
	TopLevelVar                 // Module-level variable
	LocalVar                    // Locally declared variable
	SelfVar                     // self
	FuncParamVar                // Function parameter
	StructFieldVar              // Struct field
)

type Constant struct {
	Type  Type
	Value any // TODO
}

func (c *Constant) Kind() Kind                        { return c.Type.Kind() }
func (c *Constant) Underlying() Type                  { return c.Type }
func (c *Constant) String() string                    { return "" }
func (c *Constant) StringWithName(name string) string { return "name := " + c.String() }

// Underlyer is implemented by variables and constants that have an underlying type.
type Underlyer interface {
	Type
	Underlying() Type
}
