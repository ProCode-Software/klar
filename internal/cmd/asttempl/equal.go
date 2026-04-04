package main

import "bytes"

const equalTemplate = Header + `
{{ range .}}
{{- if eq .Name "Identifier" "Operator" "BaseNode" }} {{continue}} {{end -}}
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
	{{- else if or (HasName .Type "ast.Identifier") (HasName .Type "ast.Operator") }}
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

func GenerateEqual(b *bytes.Buffer, nodes NodeList, pkg Package) error {
	return newTemplate("equal", equalTemplate).Execute(b, nodes)
}
