package codegen

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertEnum(o *analysis.Object, enum *analysis.Enum) *jsir.ClassDeclaration {
	cls := &jsir.ClassDeclaration{Name: o.Name}
	return cls
}
