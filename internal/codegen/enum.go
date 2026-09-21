package codegen

import (
	"cmp"
	"maps"
	"slices"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (g *Generator) convertEnum(o *analysis.Object, enum *analysis.Enum) *jsir.ClassDeclaration {
	stmt := o.Node().(*ast.EnumDeclaration)
	cls := &jsir.ClassDeclaration{
		Name:  o.Name,
		Flags: make(map[any]jsir.ClassPropertyFlags, len(enum.Items)),
	}
	// TODO: An inherited item may not be in the statement
	for _, itemNode := range stmt.Values {
		ei := enum.LookupItem(itemNode.Identifier.Name)
		// All enum variants are declared as static fields or methods
		if len(ei.Params) == 0 {
			binding := &jsir.Binding{Variable: &jsir.Symbol{ei.Name}}
			if itemNode.Value != nil {
				binding.Value = g.convertExpression(itemNode.Value)
			}
			cls.Fields = append(cls.Fields, binding)
			cls.Flags[binding] = jsir.Static
		} else {
			g.convertEnumFunction(ei, cls)
		}
	}
	return cls
}

func (g *Generator) convertEnumFunction(ei *analysis.EnumItem, cls *jsir.ClassDeclaration) {
	meth := &jsir.FunctionDeclaration{
		Name:   ei.Name,
		Params: make([]*jsir.FunctionParam, len(ei.Params)),
		Body:   &jsir.Block{},
	}
	var paramNames []string
	if len(ei.ParamMap) > 0 {
		paramNames = slices.SortedFunc(
			maps.Keys(ei.ParamMap), func(a, b string) int {
				return cmp.Compare(ei.ParamMap[a], ei.ParamMap[b])
			},
		)
	}
	for i := range ei.Params {
		var name string
		if paramNames != nil {
			name = paramNames[i]
		} else {
			name = generateLetter(i)
		}
		meth.Params[i] = &jsir.FunctionParam{Name: &jsir.Symbol{name}}
	}
	cls.Methods = append(cls.Methods, meth)
	cls.Flags[meth] = jsir.Static
}
