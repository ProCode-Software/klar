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

func (c *Checker) parseAttributes(attrs []ast.Statement) *Attributes {
	if len(attrs) == 0 {
		return nil
	}
	a := &Attributes{}
	for _, stmt := range attrs {
		attr := stmt.(*ast.Attribute)
		switch attr.Decorator.Name {
		default:
			// Unknown attribute name
		}
	}
	return a
}
