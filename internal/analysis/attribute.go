package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/target"
)

type Attributes struct {
	Deprecated *Deprecation
	External   []*External
	Target     target.Target
}

type Deprecation struct{}

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

func (c *Checker) parseAttributes(attrs []*ast.Attribute, kind attrTargetKind) *Attributes {
	if len(attrs) == 0 {
		return nil
	}
	a := &Attributes{}
	for _, stmt := range attrs {
		c.parseAttribute(a, stmt, kind)
	}
	return a
}

// parseAttributes2 is [Checker.parseAttributes], but takes a
// generic slice of [ast.Statement]. All elements are expected
// to have type [*ast.Attribute].
func (c *Checker) parseAttributes2(attrs []ast.Statement, kind attrTargetKind) *Attributes {
	if len(attrs) == 0 {
		return nil
	}
	a := &Attributes{}
	for _, stmt := range attrs {
		c.parseAttribute(a, stmt.(*ast.Attribute), kind)
	}
	return a
}

// parseAttribute parses a single attribute into the corresponding field in a.
func (c *Checker) parseAttribute(a *Attributes, attr *ast.Attribute, kind attrTargetKind) {
	switch attr.Decorator.Name {
	default:
		// Unknown attribute name
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
