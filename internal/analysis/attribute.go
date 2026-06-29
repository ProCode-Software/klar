package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
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

type attrTargetKind uint8

const (
	unsupportedAttribute attrTargetKind = iota

	structFieldAttribute // @name, @deprecated
	intfFieldAttribute   // @name, @deprecated, @target
	enumVariantAttribute // @name, @deprecated
	funcAttribute        // All attributes
	typeAttribute        // @name, @deprecated
	typeAliasAttribute   // @deprecated, @external
	varAttribute         // @name, @deprecated, @external
)

func (c *Checker) parseAttributes(attrs []*ast.Attribute,
	kind attrTargetKind, fid FileID,
) *Attributes {
	if len(attrs) == 0 {
		return nil
	}
	a := &Attributes{}
	for _, stmt := range attrs {
		c.parseAttribute(a, stmt, kind, fid)
	}
	return a
}

// parseAttribute parses a single attribute into the corresponding field in a.
func (c *Checker) parseAttribute(a *Attributes, attr *ast.Attribute,
	kind attrTargetKind, fid FileID,
) {
	// TODO: Should this be a limitation?
	if attributesModule == c.module {
		panic("klar._builtin.attributes module can't reference attributes")
	}
	name := attr.Name.Name
	def := attributesModule.Context.Lookup(name)
	if def == nil || !def.public || def.Kind() != KindFunction {
		// Unknown attribute
		err := klarerrs.Node(klarerrs.ErrUnknownAttribute, attr.Name)
		err.Name = name
		err.Label = "Unknown attribute " + klarerrs.Quote(name)
		c.fileError(err, fid)
		return
	}
}

func attrTargetKindOf(n ast.Node) attrTargetKind {
	switch n.(type) {
	case *ast.StructField:
		return structFieldAttribute
	case *ast.TypeAliasDeclaration:
		return typeAliasAttribute
	case *ast.EnumItem:
		return enumVariantAttribute
	case ast.TypeDeclaration:
		return typeAttribute
	case *ast.InterfaceItem:
		return intfFieldAttribute
	case *ast.FunctionDeclaration:
		return funcAttribute
	case *ast.VariableDeclaration:
		return varAttribute
	}
	return unsupportedAttribute
}
