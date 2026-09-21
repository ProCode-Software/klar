package codegen

import (
	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertStruct(o *analysis.Object, str *analysis.Struct) *jsir.ClassDeclaration {
	cls := &jsir.ClassDeclaration{
		Name:   o.Name,
		Fields: make([]*jsir.Binding, len(str.Fields)),
		// Methods: make([]*jsir.FunctionDeclaration, 0, len(str.Methods)),
	}
	for i, field := range str.Fields {
		name := field.Name
		if !field.Public {
			name = "#" + name
		}
		binding := &jsir.Binding{Variable: &jsir.Symbol{Name: name}}
		if field.Flags.Has(analysis.HasDefault) {
			// binding.Value = g.convertExpression(nil)
		}
		cls.Fields[i] = binding
	}
	return cls
}
