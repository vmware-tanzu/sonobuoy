package templates

import (
	"strings"
	"text/template"
)

// TemplateFuncs exports (currently singular) functions to be used inside the template
var TemplateFuncs = map[string]interface{}{
	"indent": func(i int, input string) string {
		split := strings.Split(input, "\n")
		ident := "\n" + strings.Repeat(" ", i)
		// Don't indent the first line, it's already indented in the template
		return strings.Join(split, ident)
	},
}

// NewTemplate declares a new template that already has TemplateFuncs in scope
func NewTemplate(name, tmpl string) *template.Template {
	return template.Must(template.New(name).Funcs(TemplateFuncs).Parse(tmpl))
}
