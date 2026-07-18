package analysis

import (
	"fmt"
	"iter"

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
		if u, ok := t.(Underlyer); ok {
			oldT := t
			if t = u.Underlying(); t != oldT {
				continue
			}
		}
		return t
	}
}

func As[T Type](t Type) T { return Underlying(t).(T) }

func UnderlyingTypeName(t Type) Type {
	for {
		if tn, ok := t.(*TypeName); ok {
			if u, ok := tn.Underlying().(Underlyer); !ok || u.Underlying() == t {
				return tn
			}
		}
		oldT := t
		if u, ok := t.(Underlyer); ok {
			if t = u.Underlying(); t == oldT {
				return t
			}
		} else {
			return t
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
				klarerrs.AddedString{Pos: node.GetRange().Start, String: "[T]("},
				klarerrs.AddedString{Pos: node.GetRange().End, String: ")"},
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
				klarerrs.AddedString{Pos: enum.Range.Start, String: "T"},
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
			klarerrs.AddedString{Pos: ut.Node.GetRange().Start, String: "T"},
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

func Walk(t Type, visit func(*Type) ast.StopCode, flags ...walkFlags) Type {
	walkInternal(&t, visit, parseFlags(flags))
	return t
}

func WalkIter(t *Type, flags ...walkFlags) iter.Seq[*Type] {
	return func(yield func(*Type) bool) {
		walkInternal(t, func(t *Type) ast.StopCode {
			if !yield(t) {
				return ast.StopWalk
			}
			return ast.ContinueWalk
		}, parseFlags(flags))
	}
}

type walkFlags uint8

const (
	walkFunction walkFlags = 1 << iota
)

func walkInternal(t *Type, visit func(*Type) ast.StopCode, flags walkFlags) ast.StopCode {
	switch code := visit(t); code {
	case ast.ContinueWalk:
	case ast.SkipParent:
		return ast.SkipChildren
	case ast.SkipList:
		return ast.SkipList
	case ast.SkipChildren:
		return ast.ContinueWalk
	case ast.StopWalk:
		return ast.StopWalk
	}
	walkGroup := func(types ...*Type) (code ast.StopCode, stop bool) {
		for _, t := range types {
			switch code := walkInternal(t, visit, flags); code {
			case ast.SkipList:
				return ast.SkipList, true
			case ast.SkipParent:
				return ast.ContinueWalk, true
			case ast.StopWalk:
				return ast.StopWalk, true
			}
		}
		return ast.ContinueWalk, false
	}
	switch t := (*t).(type) {
	case *Map:
		if code, stop := walkGroup(&t.Key, &t.Value); stop {
			return code
		}
	case *List:
		if code, stop := walkGroup(&t.Elem); stop {
			return code
		}
	case *Optional:
		if code, stop := walkGroup(&t.Elem); stop {
			return code
		}
	case *Result:
		if code, stop := walkGroup(&t.Success, &t.Error); stop {
			return code
		}
	case *Union:
		for i := range t.Types {
			switch code := walkInternal(&t.Types[i], visit, flags); code {
			case ast.SkipList:
				return ast.ContinueWalk
			case ast.SkipParent:
				return ast.ContinueWalk
			case ast.StopWalk:
				return ast.StopWalk
			}
		}
	case *Tuple:
		for i := range t.Items {
			switch code := walkInternal(&t.Items[i], visit, flags); code {
			case ast.SkipList:
				return ast.ContinueWalk
			case ast.SkipParent:
				return ast.ContinueWalk
			case ast.StopWalk:
				return ast.StopWalk
			}
		}
	case *Overload:
		if (flags & walkFunction) == 0 {
			return ast.ContinueWalk
		}
	paramLoop:
		for i := range t.Params {
			code := walkInternal(&t.Params[i].Type, visit, flags)
			switch code {
			case ast.SkipList:
				break paramLoop
			case ast.SkipParent:
				return ast.ContinueWalk
			case ast.StopWalk:
				return ast.StopWalk
			}
		}
	labelledParamLoop:
		for name := range t.labelMap {
			code := walkInternal(&t.labelMap[name].Type, visit, flags)
			switch code {
			case ast.SkipList:
				break labelledParamLoop
			case ast.SkipParent:
				return ast.ContinueWalk
			case ast.StopWalk:
				return ast.StopWalk
			}
		}
		if code, stop := walkGroup(&t.Return); stop {
			return code
		}
	case *Function:
		if (flags & walkFunction) == 0 {
			return ast.ContinueWalk
		}
	overloadLoop:
		for i := range t.Overloads {
			otype := Type(t.Overloads[i])
			code := walkInternal(&otype, visit, flags)
			if otype, ok := otype.(*Overload); ok {
				t.Overloads[i] = otype
			}
			switch code {
			case ast.SkipList:
				break overloadLoop
			case ast.SkipParent:
				return ast.ContinueWalk
			case ast.StopWalk:
				return ast.StopWalk
			}
		}
		if code, stop := walkGroup(&t.Return); stop {
			return code
		}
	case Underlyer:
		und := t.Underlying()
		if und == t {
			break
		}
		// TODO: This doesn't actually mutate the underlying type
		if code, stop := walkGroup(&und); stop {
			return code
		}
	}
	return ast.ContinueWalk
}

func Substitute(t Type, subMap map[Type]Type) Type {
	if subMap == nil {
		return t
	}
	t = Walk(t, func(t *Type) ast.StopCode {
		if rep, ok := subMap[*t]; ok {
			*t = rep
		}
		return ast.ContinueWalk
	}, walkFunction)
	return t
}
