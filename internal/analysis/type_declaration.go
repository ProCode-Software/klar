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
		tag := &TagType{Implements: make(map[Type]struct{}, len(node.InheritedTypes))}
		existingMap := make(map[Type]ranges.Range, len(node.InheritedTypes))
		for _, tn := range node.InheritedTypes {
			typ := c.parseType(tn, decl.file)
			if _, ok := tag.Implements[typ]; ok {
				// Type specified twice
				err := klarerrs.Range(klarerrs.ErrDuplicateInheritedType, tn.GetRange())
				err.Highlights = append(err.Highlights, klarerrs.Highlight{
					Range:   existingMap[typ],
					Message: "It was already specified here",
				})
				c.fileError(err, o.file)
				continue
			}
			if !c.validateInheritedType(tn, typ, KindTag, o.file) {
				continue
			}
			tag.Implements[typ] = struct{}{}
			existingMap[typ] = tn.GetRange()
		}
		o.typ.(*TypeName).Type = tag
	case *ast.InterfaceDeclaration:
		c.checkInterfaceDecl(o, node, decl.file)
	default:
		panic(fmt.Sprintf("unknown type declaration: %T", node))
	}
}

// validateInheritedType checks the inherited type represented by node n
// and type t for validity as an inherited type for declaration kind declKind.
func (c *Checker) validateInheritedType(n ast.Type, t Type,
	declKind Kind, fid FileID,
) bool {
	kind := t.Kind()
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
	switch n.(type) {
	case *ast.TypeAlias, *ast.PrimitiveType, *ast.GenericType:
	default:
		// Change the kind so an error is reported below
		kind = InvalidType
	}
	// Validate the actual type
	switch declKind {
	case KindTag:
		if kind != KindTag {
			newError("A tag", "another tags")
			return false
		}
	case KindStruct:
		if kind != KindInterface && kind != KindStruct && kind != KindTag {
			newError("A struct", "an interface, tag, or another struct")
			return false
		}
	case KindInterface:
		if kind != KindInterface && kind != KindStruct {
			newError("An interface", "a struct or another interface")
			return false
		}
	case KindEnum:
		if kind != KindInterface && kind != KindEnum && kind != KindTag {
			newError("An enum", "an interface, tag, or another enum")
			return false
		}
	default:
		panic(fmt.Sprintf("invalid target type kind %d", declKind))
	}
	return true
}
