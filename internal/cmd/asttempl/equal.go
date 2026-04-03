package main

import (
	"go/types"
)

const equalTemplate = Header + `
{{- range .}}
{{- if eq .Name "Identifier" }} {{continue}} {{end-}}
func (a *{{.Name}}) Equal(b2 Node) bool {
	b, ok := b2.(*{{.Name}})
	if !ok {
		return false
	}
	if a == nil || b == nil {
		return a == b
	}
	{{- range (ToStruct .).Fields}}
	{{- if eq .Name "BaseNode"}}
	{{- else if HasName .Type "ast.Identifier" }}
	if !a.{{.Name}}.Equal(b.{{.Name}}) {
		return false
	}
	{{- else if IsNode .Type}}
	if a.{{.Name}} != nil && b.{{.Name}} != nil && !a.{{.Name}}.Equal(b.{{.Name}}) {
		return false
	}
	{{- else if IsSlice .Type}}
	if !equalSlice(a.{{.Name}}, b.{{.Name}}) {
		return false
	}
	{{- else}}
	if a.{{.Name}} != b.{{.Name}} {
		return false
	}
	{{- end}}
	{{- end}}
	return true
}
{{end -}}
`

func GenerateEqual(nodes NodeList, pkg Package) error {
	for _, node := range nodes {
		str, _ := node.Type().Underlying().(*types.Struct)
		for field := range str.Fields() {
			_=field
			// check if field's type is Identifier
			
			continue
			// isSlice := field.Type().(*types.Slice)
		}
	}
	return newTemplate("equal", equalTemplate).Execute(getFile("equal.go"), nodes)
}
