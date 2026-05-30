package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// TagType represents a Klar tag type.
type TagType struct{ Implements map[Type]struct{} }

func (TagType) Kind() Kind { return KindTag }

// checkCustomTypeDecl checks the type declaration in decl.node and sets
// the type of o's Type. o's Type should be [*TypeName]. The completed
// declaration is created inside the [*TypeName].
func (c *Checker) checkCustomTypeDecl(o *Object, decl *DeclarationInfo) {
	node := decl.node.(ast.TypeDeclaration)
	switch node := node.(type) {
	case *ast.StructDeclaration:
		c.checkStructDecl(o, node, decl.file)
	case *ast.EnumDeclaration:
		c.checkEnumDecl(o, node, decl.file)
	case *ast.TypeAliasDeclaration:
		c.checkTypeAlias(o, node, decl.file)
	case *ast.TagDeclaration:
		c.checkTagType(o, decl)
	case *ast.InterfaceDeclaration:
		c.checkInterfaceDecl(o, node, decl.file)
	default:
		panic(fmt.Sprintf("unknown type declaration: %T", node))
	}
}

func (c *Checker) checkTagType(o *Object, decl *DeclarationInfo) {
	node := decl.node.(*ast.TagDeclaration)
	o.typ.(*TypeName).Type = &TagType{
		Implements: c.checkInheritedTypes(node.InheritedTypes, KindTag, o.file, decl.file),
	}
}

// checkInheritedTypes checks the inherited types of a tag declaration and
// returns a map of the types that are inherited. Errors are reported for
// undefined and duplicate types.
func (c *Checker) checkInheritedTypes(names []ast.Type, kind Kind,
	fid FileID, ctx *Context,
) (inherited map[Type]struct{}) {
	inherited = make(map[Type]struct{}, len(names))
	existingMap := make(map[Type]ranges.Range, len(names))
	for _, tn := range names {
		typ := c.parseType(tn, ctx)
		if _, ok := inherited[typ]; ok {
			// Type specified twice
			err := klarerrs.Range(klarerrs.ErrDuplicateInheritedType, tn.GetRange())
			err.Highlights = append(err.Highlights, klarerrs.Highlight{
				Range:   existingMap[typ],
				Message: "It was already specified here",
			})
			c.fileError(err, fid)
			continue
		}
		if !c.validateInheritedType(tn, typ, kind, fid) {
			continue
		}
		inherited[typ] = struct{}{}
		existingMap[typ] = tn.GetRange()
	}
	return inherited
}

// validateInheritedType checks the inherited type represented by node n
// and type t for validity as an inherited type for declaration kind declKind.
func (c *Checker) validateInheritedType(n ast.Type, t Type,
	expKind Kind, fid FileID,
) bool {
	gotKind := t.Kind()
	// Validate the node
	newError := func(currType, allowedTypes string) {
		err := klarerrs.Range(klarerrs.ErrInvalidInheritedType, n.GetRange())
		err.Params = klarerrs.ErrorParams{
			"kind":         currType,
			"allowedTypes": allowedTypes,
		}
		err.Label = "Can't inherit from this kind of type"
		c.fileError(err, fid)
	}
	if expKind == gotKind || gotKind == InvalidType {
		return true
	}
	switch n.(type) {
	case *ast.TypeAlias, *ast.PrimitiveType, *ast.GenericType:
	default:
		// Change the kind so an error is reported below
		gotKind = InvalidType
	}
	// Validate the actual type
	switch expKind {
	case KindTag:
		if gotKind != KindTag {
			newError("A tag", "another tag")
			return false
		}
	case KindStruct:
		if gotKind != KindInterface && gotKind != KindStruct && gotKind != KindTag {
			newError("A struct", "an interface, tag, or another struct")
			return false
		}
	case KindInterface:
		if gotKind != KindInterface && gotKind != KindStruct {
			newError("An interface", "a struct or another interface")
			return false
		}
	case KindEnum:
		if gotKind != KindInterface && gotKind != KindEnum && gotKind != KindTag {
			newError("An enum", "an interface, tag, or another enum")
			return false
		}
	default:
		panic(fmt.Sprintf("invalid target type kind %d", expKind))
	}
	return true
}
