package generator

import (
	"strings"
	"text/template"
)

const defaultStructTemplate = `type {{ .Name }} struct {
{{- range .Fields }}
{{- if .Comment }}
	// {{ .Comment }}
{{- end }}
	{{ .Name }} {{ .FieldType }}{{ if .Tag }} ` + "`" + `lisp-rpc:"{{ .Tag }}"` + "`" + `{{ end }}
{{- end }}
}`

const defaultRPCImplTemplate = `func ({{ .Name }}) ToRPCType() server.RPCType {
{{- if eq .RPCType.String "msg" }}
	return server.NewRPCTypeMsg("{{ .DataName }}")
{{- else if eq .RPCType.String "rpc" }}
	return server.NewRPCTypeRPC("{{ .DataName }}")
{{- else if eq .RPCType.String "map" }}
	return server.NewRPCTypeMap()
{{- else if eq .RPCType.String "list" }}
	return server.NewRPCTypeList()
{{- else }}
	return server.NewRPCTypeV()
{{- end }}
}
{{- if and (eq .RPCType.String "rpc") .ReturnType }}

func ({{ .Name }}) ReturnType() {{ .ReturnType }} {
	return {{ .ReturnType }}{}
}
{{- end }}`

const defaultInitTemplate = `func init() {
{{- range .MapTypes }}
	server.RegisterGlobalMapType("{{ . }}")
{{- end }}
}`

const defaultPackageTemplate = `module {{ .PkgName }}

go 1.22

require github.com/ccqpein/lisp-rpc-golang v0.0.0`

const defaultHeaderTemplate = `package {{ .PackageName }}

import (
	"github.com/ccqpein/lisp-rpc-golang/server"
)`

// DefaultStructTemplate returns the default text template for generating Go structs.
func DefaultStructTemplate() string {
	return defaultStructTemplate
}

// DefaultRPCImplTemplate returns the default template for generating ToRPCType and ReturnType methods.
func DefaultRPCImplTemplate() string {
	return defaultRPCImplTemplate
}

// DefaultInitTemplate returns the default template for generating the package init() function.
func DefaultInitTemplate() string {
	return defaultInitTemplate
}

// DefaultPackageTemplate returns the default template for generating go.mod.
func DefaultPackageTemplate() string {
	return defaultPackageTemplate
}

// DefaultHeaderTemplate returns the default template for the Go file header.
func DefaultHeaderTemplate() string {
	return defaultHeaderTemplate
}

// RenderTemplate renders a text template with the given data.
func RenderTemplate(tmplStr string, data any) (string, error) {
	tmpl, err := template.New("tmpl").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}

	return sb.String(), nil
}
