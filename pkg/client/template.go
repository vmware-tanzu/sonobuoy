/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/vmware-tanzu/sonobuoy/pkg/types"
)

// TemplateFuncs exports (currently singular) functions to be used inside the template
var (
	templateFuncs = map[string]interface{}{
		"indent": func(i int, input string) string {
			split := strings.Split(input, "\n")
			ident := "\n" + strings.Repeat(" ", i)
			// Don't indent the first line, it's already indented in the template
			return strings.Join(split, ident)
		},
	}

	//go:embed gen.tmpl.yaml
	templateDoc string

	// genManifest is the template for the `sonobuoy gen` output
	genManifest = newTemplate("manifest", templateDoc)
)

// secContextFromMode turns a simple string "mode" into the security context it refers to. Users could
// completely customize this by using 'sonobuoy gen' and editing it, but this provides a fast/easy way
// to switch between common values.
func secContextFromMode(mode types.SecurityContextMode) string {
	// TODO(jschnake): Seems like we should be using an actual object and marshalling it
	// but we get into version issues (at time of writing this fsgroup is a new, beta feature).
	// Just explicitly writing it for now and we can evolve this if other use cases come up.
	switch mode {
	case types.SecurityContextModeNone:
		return ""
	case types.SecurityContextModeNonRoot:
		return "securityContext:\n    runAsUser: 1000\n    runAsGroup: 3000\n    fsGroup: 2000"
	default:
		return "securityContext:\n    runAsUser: 1000\n    runAsGroup: 3000\n    fsGroup: 2000"
	}
}

// newTemplate declares a new template that already has templateFuncs in scope
func newTemplate(name, tmpl string) *template.Template {
	return template.Must(template.New(name).Funcs(templateFuncs).Parse(tmpl))
}
