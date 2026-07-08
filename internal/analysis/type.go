package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

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
		return typ // Already typed
		// TODO: This could be list of untyped. Walk the types and run toTyped
	}
}

func Walk(t Type, visit func(Type) ast.StopCode) {
	walkInternal(t, visit)
}

func walkInternal(t Type, visit func(Type) ast.StopCode) bool {
	_ = visit(t)
	switch t := t.(type) {
	case Kind:

	default:
		panic(fmt.Sprintf("Walk: unhandled type: %T", t))
	}
	return true
}
