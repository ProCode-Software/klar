package command

var fullHelpTemplate = `
{{- .ArgUsage -}}
{{ if .Aliases }}
{{ .AliasesString }}
{{- end }}

{{ or .LongDescription .ShortDescription | wrap }}

{{ if and .Flags (len .Flags.FlagDefs) -}}
{{ title "Flags" }}
{{ .FlagString 2 -}}
{{- end -}}

{{- if .Examples }}
{{ title "Examples" -}}
{{ range .Examples }}
  {{ printf "%s:" .Description | ansi "2" }}
  {{ exec }} {{ bold "33" .Command }} 
  		{{- range .Args }} {{ if hasPrefix . "." -}}
    		{{ ansi "34" . }}
     	{{- else -}} {{ . }} {{- end}}
    {{- end }}
	{{- range .Flags }} {{ if hasPrefix . "-" -}} {{ ansi "36" . }} 
		{{- else -}} {{ ansi "32" . }} {{- end -}}
	{{ end }}
{{ end }}
{{ end -}}

{{- if .SeeAlso -}}
{{ title "See also" }}
{{ .SeeAlsoString 2 }}
{{- end -}}`

var usageTemplate = `
{{- title "Usage" -}} {{ exec }} {{ bold "33" .Name }}
{{- range .Usage }} {{ ansi "36" . }} {{- end -}}`
