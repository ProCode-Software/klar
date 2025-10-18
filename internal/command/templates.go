package command

var fullHelpTemplate = `
{{- $root := . -}}
{{.ArgUsage}}

{{or .LongDescription .ShortDescription -}}

{{- if .Examples }}

{{.Title "Examples" -}}

{{range .Examples }}

  {{.Description}}:
  {{$root.FormatExecName }} {{$root.Bold "33" .Command }}
	{{- if .Args }} {{ join .Args " "}} {{- end}}
	{{- range .Flags }} {{if eq (index . 0) '-' -}}
		{{ $root.ANSI "36" . }} 
	{{- else -}}
		{{ $root.ANSI "34" . }}
	{{- end -}}
	{{end}}
{{- end -}}
{{- end -}}

{{- if gt (len .SeeAlso) 0}}

{{.Title "See also" }}
{{.SeeAlsoString 2}}
{{- end}}`
