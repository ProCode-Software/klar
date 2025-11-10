package command

var fullHelpTemplate = `
{{- .ArgUsage}}

{{ or .LongDescription .ShortDescription -}}

{{- if and .Flags (len .Flags.FlagDefinitions) }}

{{ title "Flags" }}
{{ .FlagString 2 }}
{{- end -}}

{{- if .Examples }}

{{ title "Examples" -}}
{{ range .Examples }}
  {{ printf "%s:" .Description | ansi "2" }}
  {{ exec }} {{ bold "33" .Command }} {{- if .Args }} {{ join .Args " " }} {{- end }}
	{{- range .Flags }} {{ if hasPrefix . "-" -}} {{ ansi "36" . }} 
		{{- else -}} {{ ansi "34" . }} {{- end -}}
	{{ end }}
{{ end -}}

{{- end -}}

{{- if .SeeAlso }}
{{ title "See also" }}
{{ .SeeAlsoString 2 }}
{{- end -}}`

var usageTemplate = `
{{- title "Usage" -}} {{ exec }} {{ bold "33" .Name }}
{{- range .Usage }} {{ ansi "36" . }} {{- end -}}`
