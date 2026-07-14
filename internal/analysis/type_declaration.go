package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// TypeName represents a type declaration.
//
// Type is one of these types:
//   - [*TypeAlias]
//   - [*Struct]
//   - [*Interface]
//   - [*Enum]
//   - [*Tag]
type TypeName struct {
	Type
	Name string
}

// String returns the name of the type.
func (n *TypeName) String() string {
	if alias, ok := n.Type.(*TypeAlias); ok {
		return alias.Type.String()
	}
	return n.Name
}
func (n *TypeName) Underlying() Type { return n.Type }
func (n *TypeName) objKind()         {}

// Alias
// =======

type TypeAlias struct {
	Name string
	Type
	resolved Type
}

func (a *TypeAlias) Resolve() Type {
	if a.resolved == nil {
		if subAlias, ok := a.Type.(*TypeAlias); ok {
			// This also recursively resolves aliases for upstream type aliases
			a.resolved = subAlias.Resolve()
		} else {
			// a's underlying type isn't an alias, so it is already resolved
			a.resolved = a.Type
		}
	}
	return a.resolved
}

func (a *TypeAlias) Kind() Kind       { return a.Resolve().Kind() }
func (a *TypeAlias) Underlying() Type { return a.Resolve() }

func (c *Checker) checkTypeAlias(o *Object, node *ast.TypeAliasDeclaration) {
	alias := &TypeAlias{Name: o.name}
	o.TypeName().Type = alias
	rhs := c.parseType(node.Type, o.LookupContext())
	// Set to invalid if we couldn't typecheck the rhs
	if rhs == nil {
		rhs = InvalidType
	}
	if rhs.Kind() == KindGeneric {
		// The target of a type alias cannot be a generic type
		err := klarerrs.Range(klarerrs.ErrGenericTypeAlias, node.Type.GetRange())
		err.Label = "This can't be a generic type"
		c.fileError(err, o.file)
		rhs = InvalidType
	}
	alias.Type = rhs
}

func (c *Checker) checkFuncAlias(o *Object) {
	decl := o.info
	targetExpr := decl.node.(*ast.FuncAliasDeclaration).Target
	// TODO: Lookup the target expression and make sure it resolves to a function
	var target *Object = nil

	_ = targetExpr
	o.typ.(*FunctionAlias).Target = target
}

func Unalias(t Type) Type {
	if a, ok := t.(*TypeAlias); ok {
		return a.Resolve()
	}
	return t
}

// Tag
// ======

// Tag represents a Klar tag type.
type Tag struct{ Implements map[Type]struct{} }

func (*Tag) Kind() Kind     { return KindTag }
func (*Tag) String() string { return "<tag>" }

// checkTypeDecl checks the type declaration in decl.node and sets
// the type of o's Type. o's Type should be [*TypeName]. The completed
// declaration is created inside the [*TypeName].
func (c *Checker) checkTypeDecl(o *Object) {
	node := o.info.node.(ast.TypeDeclaration)
	_ = o.TypeName() // Should be a [*TypeName]
	switch node := node.(type) {
	case *ast.StructDeclaration:
		c.checkStructDecl(o, node)
	case *ast.EnumDeclaration:
		c.checkEnumDecl(o, node)
	case *ast.TypeAliasDeclaration:
		c.checkTypeAlias(o, node)
	case *ast.TagDeclaration:
		c.checkTagType(o, node)
	case *ast.InterfaceDeclaration:
		c.checkInterfaceDecl(o, node)
	default:
		panic(fmt.Sprintf("unknown type declaration: %T", node))
	}
}

func (c *Checker) checkTagType(o *Object, node *ast.TagDeclaration) {
	// TODO: Check that each inherited type was declared within this module
	o.TypeName().Type = &Tag{
		Implements: c.checkInheritedTypes(node.InheritedTypes, KindTag, o.LookupContext()),
	}
}

// checkInheritedTypes checks the inherited types to ensure they are
// compatible with the given target declaration kind.
func (c *Checker) checkInheritedTypes(
	names []ast.Type, kind Kind, fctx *Context,
) (inherited map[Type]struct{}) {
	if len(names) == 0 {
		return nil
	}
	inherited = make(map[Type]struct{}, len(names))
	existingMap := make(map[Type]ranges.Range, len(names))
	for _, tn := range names {
		var flags Flag
		// Tags can only inherit from locally-declared tags
		if kind == KindTag {
			flags |= LocalOnly
		}
		typ := c.parseType(tn, fctx, flags)
		underlying := Underlying(typ)
		if typ.Kind() == InvalidType {
			continue
		}
		if _, ok := inherited[underlying]; ok {
			// Type specified twice
			err := klarerrs.Range(klarerrs.ErrDuplicateInheritedType, tn.GetRange())
			err.Name = typ.String()
			err.Highlights = append(err.Highlights, klarerrs.Highlight{
				Range:   existingMap[underlying],
				Message: "It was already specified here",
			})
			c.fileError(err, fctx.File)
			continue
		}
		if !c.validateInheritedType(tn, typ, kind, fctx.File) {
			continue
		}
		inherited[underlying] = struct{}{}
		existingMap[underlying] = tn.GetRange()
	}
	return inherited
}

// validateInheritedType checks the inherited type represented by node n
// and type t for validity as an inherited type for declaration kind declKind.
func (c *Checker) validateInheritedType(n ast.Type, t Type,
	targetKind Kind, fid FileID,
) bool {
	// TODO: Maybe we should allow inheriting from primitive types (not lists)
	typeKind := t.Kind()
	// Validate the node
	newError := func(currType, allowedTypes string) {
		err := klarerrs.Range(klarerrs.ErrInvalidInheritedType, n.GetRange())
		err.Params = klarerrs.ErrorParams{"kind": currType, "allowedTypes": allowedTypes}
		err.Label = "Can't inherit from this kind of type"
		c.fileError(err, fid)
	}
	if targetKind == typeKind || typeKind == InvalidType {
		return true
	}
	switch n.(type) {
	case *ast.TypeAlias, *ast.PrimitiveType, *ast.GenericType:
	default:
		// Change the kind so an error is reported below
		typeKind = InvalidType
	}
	// Validate the actual type
	switch targetKind {
	case KindTag:
		// Already checked via targetKind == typeKind, so this is invalid
		newError("A tag", "another tag")
		return false
	case KindStruct:
		if typeKind != KindInterface && typeKind != KindTag {
			newError("A struct", "an interface, tag, or another struct")
			return false
		}
	case KindInterface:
		if typeKind != KindStruct {
			newError("An interface", "a struct or another interface")
			return false
		}
	case KindEnum:
		if typeKind != KindInterface && typeKind != KindTag {
			newError("An enum", "an interface, tag, or another enum")
			return false
		}
	default:
		panic(fmt.Sprintf("invalid target type kind %d", targetKind))
	}
	return true
}

/*
checkDirectCycles checks for direct cycles within the declarations in the
given context. Type aliases that directly refer to other type aliases are checked.

The algorithm for this is similar to [Checker.objPathIndex] and the one used
in Go's type checker:

  - Not found in `pathI`: The type name hasn't been seen before (red)
  - Found in `pathI` with a value >= 0: Has been seen but is not done (white)
  - Value < 0: Seen and done (blue)
*/
func (c *Checker) checkDirectCycles(ctx *Context) {
	pathI := make(map[*Object]int)
	for _, obj := range ctx.SortedDecls() {
		if !obj.IsTypeName() {
			continue
		}
		var path []*Object
		for {
			if start, ok := pathI[obj]; start < 0 {
				break // Object is blue
			} else if ok {
				// Object is white: there is a cycle starting at `path[start]`
				obj.TypeName().Type = InvalidType
				c.error(cycleError(path[start:]))
				break
			}
			// Object is red
			pathI[obj] = len(path)
			path = append(path, obj)

			// If this object isn't a type alias, we're at the end of the path and done.
			aliasDecl, ok := obj.info.node.(*ast.TypeAliasDeclaration)
			if !ok {
				break
			}
			// TODO: We should also check for cycles in parenthesized types, tuples,
			// and types with generics.
			//
			// Go only checks in aliases because recursive types are allowed in other
			// wrapper types (such as pointers). `type T *T` is even allowed in Go.
			// Though in this step, Go doesn't check inside structs.
			var rhs *ast.TypeAlias
			rhs, ok = aliasDecl.Type.(*ast.TypeAlias)
			if !ok {
				break
			}

			// Resolve the type the alias refers to. [Object.IsTypeName] handles nil
			// objects. In that case, if the RHS is undefined, an error will be
			// reported later. Also, we're not looking up recursively -- types can't be
			// declared across contexts/scopes. Similar for imported types. In those
			// cases, we can stop.
			next := ctx.Lookup(rhs.Identifier)
			if !next.IsTypeName() {
				break
			}
			obj = next
		}
		// Mark all type names in the path blue
		for _, obj := range path {
			pathI[obj] = -1
		}
	}
}
