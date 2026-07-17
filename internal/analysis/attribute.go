package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

// Contains the definitions of attributes
var attributesModule *Module

// All fields are optional
type Attributes struct {
	Deprecated *Deprecation
	External   []*External
	Target     []target.Target
	Added      *version.Version
	Name       map[target.Target]string
}

// All fields are optional
type Deprecation struct {
	Reason string
	Since  *version.Version
	Use    string // What users should use instead
	// Specific targets this is deprecated on. If not specified,
	// the deprecation applies to all targets.
	On []target.Target
}

type External struct{}

type attrMode uint8

const (
	// @added and @deprecated are allowed on all declarations
	nameAttr attrMode = 1 << iota
	targetAttr
	externalAttr

	enumVariantAttrs = nameAttr
	funcAliasAttrs   = nameAttr
	funcAttrs        = nameAttr | targetAttr | externalAttr
	intfAttrs        = 0 // @deprecated and @added only
	intfFieldAttrs   = nameAttr | targetAttr
	structFieldAttrs = nameAttr
	typeAttrs        = nameAttr
	tagAttrs         = 0
	typeAliasAttrs   = externalAttr
	varAttrs         = nameAttr | externalAttr
)

func (c *Checker) parseAttributes(attrs []*ast.Attribute,
	target attrTarget, nodeRange ranges.Range, fid FileID,
) *Attributes {
	if len(attrs) == 0 {
		return nil
	}
	a := &Attributes{}
	for _, stmt := range attrs {
		c.parseAttribute(a, stmt, target, nodeRange, fid)
	}
	return a
}

// parseAttribute parses a single attribute into the corresponding field in a.
func (c *Checker) parseAttribute(a *Attributes, attr *ast.Attribute,
	t attrTarget, nodeRange ranges.Range, fid FileID,
) {
	// TODO: Should this be a limitation?
	if attributesModule == c.module {
		panic("klar._builtin.attributes module can't reference attributes")
	}
	name := attr.Name.Name
	def := attributesModule.Context.Lookup(name)
	if def == nil || !def.Public || def.Kind() != KindFunction {
		// Unknown attribute
		err := klarerrs.Node(klarerrs.ErrUnknownAttribute, attr.Name)
		err.Name = name
		err.Label = "Unknown attribute " + klarerrs.Quote(name)
		c.fileError(err, fid)
		return
	}
	// Check if the attribute is supported on the current declaration type
	if mode := map[string]attrMode{
		"name": nameAttr, "target": targetAttr, "external": externalAttr,
		// Attributes not in this map are supported on all declarations
	}[name]; mode != 0 && !t.supports(mode) {
	}
}

type attrTarget struct {
	node   ast.Node
	str    string
	mode   attrMode
	public bool
}

func (t attrTarget) supports(m attrMode) bool { return (t.mode & m) != 0 }

func attrTargetKindOf(n ast.Node, public bool) attrTarget {
	switch n.(type) {
	// Other declarations
	case *ast.FunctionDeclaration:
		return attrTarget{n, "a function", funcAttrs, public}
	case *ast.VariableDeclaration:
		return attrTarget{n, "a variable", varAttrs, public}
	case *ast.FuncAliasDeclaration:
		return attrTarget{n, "a function alias", funcAliasAttrs, public}

	// Type declarations
	case *ast.TypeAliasDeclaration:
		return attrTarget{n, "a type alias", typeAliasAttrs, public}
	case *ast.InterfaceDeclaration:
		return attrTarget{n, "an interface", intfAttrs, public}
	case *ast.StructDeclaration, *ast.EnumDeclaration:
		return attrTarget{n, "a type", typeAttrs, public}
	case *ast.TagDeclaration:
		return attrTarget{n, "a tag", tagAttrs, public}

	// Entries in type declarations
	case *ast.EnumItem:
		return attrTarget{n, "an enum variant", enumVariantAttrs, public}
	case *ast.InterfaceItem:
		return attrTarget{n, "an interface field", intfFieldAttrs, public}
	case *ast.StructField:
		return attrTarget{n, "a struct field", structFieldAttrs, public}
	}
	panic(fmt.Sprintf("unhandled or unsupported attribute target: %T", n))
	// return unsupportedAttr
}
